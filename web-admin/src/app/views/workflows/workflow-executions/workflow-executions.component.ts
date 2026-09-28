import { Component, OnInit, ViewChild } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import {
  TableModule,
  ButtonModule,
  CardModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ModalModule,
  FormModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition, WorkflowExecution, WorkflowExecutionDetail, WaitingEvent, WaitingEventFormField } from '../services/workflows.service';
import { UsersService } from '../../users/services/users.service';
import { ErrorUtil } from '../../../shared/utils/error.util';
import { BpmnDesignerComponent, DEFAULT_BPMN_XML } from '../../../shared/components/bpmn-designer/bpmn-designer.component';

@Component({
  selector: 'app-workflow-executions',
  templateUrl: './workflow-executions.component.html',
  styleUrls: ['./workflow-executions.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    TableModule,
    ButtonModule,
    CardModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    ModalModule,
    FormModule,
    IconDirective,
    BpmnDesignerComponent
  ]
})
export class WorkflowExecutionsComponent implements OnInit {
  workflowId: string = '';
  executionIdParam: string = '';
  scope: 'global' | 'tenant' = 'global';
  tenantCode: string = '';

  workflow: WorkflowDefinition | null = null;
  loadingWorkflow: boolean = false;

  executionsList: WorkflowExecution[] = [];
  loadingExecutions: boolean = false;
  executionsErrorMessage: string = '';
  executionStatusFilter: string = '';

  showExecutionDetailModal: boolean = false;
  selectedExecutionDetail: WorkflowExecutionDetail | null = null;
  selectedExecutionId: string = '';
  selectedExecutionUserName: string = '';
  loadingExecutionDetail: boolean = false;
  executionDetailError: string = '';
  activeDetailTab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'waitingEvents' | 'diagram' = 'overview';

  waitingEvents: WaitingEvent[] = [];
  eventFormData: Record<string, Record<string, any>> = {};
  eventRawJson: Record<string, string> = {};
  eventFormMode: Record<string, 'form' | 'json'> = {};
  submittingWaitEvent: Record<string, boolean> = {};
  waitEventErrors: Record<string, string> = {};
  waitEventSuccess: Record<string, string> = {};
  fieldValidationErrors: Record<string, Record<string, string>> = {};

  @ViewChild('executionBpmnDesigner') executionBpmnDesigner?: BpmnDesignerComponent;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private location: Location,
    private workflowsService: WorkflowsService,
    private usersService: UsersService
  ) {}

  ngOnInit(): void {
    this.route.queryParams.subscribe(queryParams => {
      if (queryParams['tenantCode']) {
        this.tenantCode = queryParams['tenantCode'];
        this.scope = 'tenant';
      } else if (queryParams['scope']) {
        this.scope = queryParams['scope'];
      }
    });

    this.route.params.subscribe(params => {
      if (params['workflowId']) {
        this.workflowId = params['workflowId'];
        this.loadWorkflow(this.workflowId);
        this.loadExecutions();

        if (params['executionId']) {
          this.selectedExecutionId = params['executionId'];
          this.openExecutionDetail(this.selectedExecutionId);
        }
      }
    });
  }

  loadWorkflow(id: string): void {
    this.loadingWorkflow = true;
    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflow(id)
      : this.workflowsService.getTenantWorkflow(this.tenantCode, id);

    req$.subscribe({
      next: (wf: any) => {
        const entity = wf.Entity || wf.entity || wf;
        this.workflow = {
          id: entity.id || entity.ID || '',
          code: entity.code || entity.Code || '',
          vnamespace: entity.vnamespace || entity.VNamespace || '',
          name: entity.name || entity.Name || entity.code || '',
          description: entity.description || entity.Description || '',
          version: entity.version || entity.Version || 1,
          onVersionChange: entity.onVersionChange || entity.OnVersionChange || 'defined_in_execution',
          payload: entity.payload || entity.Payload || '',
          payloadFormat: (entity.payloadFormat || entity.PayloadFormat || 'json').toLowerCase() as any,
          maxDurationSeconds: entity.maxDurationSeconds || 0,
          isActive: entity.isActive !== undefined ? entity.isActive : true,
          scope: this.scope,
          userId: entity.userId || entity.UserID || '',
          userName: entity.userName || entity.UserName || '',
          accountId: entity.accountId || entity.AccountID || '',
          accountName: entity.accountName || entity.AccountName || '',
          externalUserId: entity.externalUserId || entity.ExternalUserID || ''
        };
        this.loadingWorkflow = false;
      },
      error: () => {
        this.loadingWorkflow = false;
      }
    });
  }

  loadExecutions(): void {
    if (!this.workflowId) return;
    this.loadingExecutions = true;
    this.executionsErrorMessage = '';

    const vns = this.workflow?.vnamespace || '';
    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalExecutions(this.workflowId, this.executionStatusFilter, 50, '', vns)
      : this.workflowsService.getTenantExecutions(this.tenantCode, this.workflowId, this.executionStatusFilter, 50, '', vns);

    req$.subscribe({
      next: (res: any) => {
        const raw = res.Entities || res.entities || [];
        this.executionsList = raw.map((e: any) => ({
          id: e.id || e.ID || '',
          workflowDefinitionId: e.workflowDefinitionId || e.WorkflowDefinitionID || '',
          workflowDefinitionVersion: e.workflowDefinitionVersion || e.WorkflowDefinitionVersion || 1,
          vnamespace: e.vnamespace || e.VNamespace || '',
          executionKey: e.executionKey || e.ExecutionKey || '',
          status: (e.status || e.Status || 'pending').toLowerCase(),
          input: e.input || e.Input || null,
          output: e.output || e.Output || null,
          stateData: e.stateData || e.StateData || null,
          error: e.error || e.Error || '',
          userId: e.userId || e.UserID || '',
          userName: e.userName || e.UserName || '',
          accountId: e.accountId || e.AccountID || '',
          accountName: e.accountName || e.AccountName || '',
          externalUserId: e.externalUserId || e.ExternalUserID || '',
          startedAt: e.startedAt || e.StartedAt || null,
          completedAt: e.completedAt || e.CompletedAt || null,
          createdAt: e.createdAt || e.CreatedAt || '',
          updatedAt: e.updatedAt || e.UpdatedAt || ''
        }));

        const userIds = this.executionsList.map(x => x.userId).filter((id): id is string => !!id);
        if (userIds.length > 0) {
          this.usersService.resolveUsers(userIds).subscribe(userMap => {
            this.executionsList.forEach(x => {
              if (x.userId && userMap.has(x.userId)) {
                x.userName = userMap.get(x.userId);
              }
            });
          });
        }

        this.loadingExecutions = false;
      },
      error: (err) => {
        this.executionsErrorMessage = ErrorUtil.formatErrorMessage(err);
        this.loadingExecutions = false;
      }
    });
  }

  openExecutionDetail(executionId: string): void {
    this.selectedExecutionId = executionId;
    this.showExecutionDetailModal = true;
    this.loadingExecutionDetail = true;
    this.executionDetailError = '';
    this.selectedExecutionDetail = null;
    this.selectedExecutionUserName = '';

    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalExecutionDetail(executionId)
      : this.workflowsService.getTenantExecutionDetail(this.tenantCode, executionId);

    req$.subscribe({
      next: (res: any) => {
        const detailObj = res?.Result || res?.result || res;
        const rawExec = detailObj?.execution || detailObj?.Execution || detailObj;
        const rawTokens = detailObj?.tokens || detailObj?.Tokens || [];
        const rawJobs = detailObj?.jobs || detailObj?.Jobs || [];

        this.selectedExecutionDetail = {
          execution: {
            id: rawExec.id || rawExec.ID || '',
            workflowDefinitionId: rawExec.workflowDefinitionId || rawExec.WorkflowDefinitionID || '',
            workflowDefinitionVersion: rawExec.workflowDefinitionVersion || rawExec.WorkflowDefinitionVersion || 1,
            payloadSnapshot: rawExec.payloadSnapshot || rawExec.PayloadSnapshot || '',
            vnamespace: rawExec.vnamespace || rawExec.VNamespace || '',
            executionKey: rawExec.executionKey || rawExec.ExecutionKey || '',
            status: (rawExec.status || rawExec.Status || 'pending').toLowerCase(),
            input: rawExec.input || rawExec.Input || null,
            output: rawExec.output || rawExec.Output || null,
            stateData: rawExec.stateData || rawExec.StateData || null,
            error: rawExec.error || rawExec.Error || '',
            userId: rawExec.userId || rawExec.UserID || '',
            userName: rawExec.userName || rawExec.UserName || '',
            accountId: rawExec.accountId || rawExec.AccountID || '',
            accountName: rawExec.accountName || rawExec.AccountName || '',
            externalUserId: rawExec.externalUserId || rawExec.ExternalUserID || '',
            startedAt: rawExec.startedAt || rawExec.StartedAt || null,
            completedAt: rawExec.completedAt || rawExec.CompletedAt || null,
            createdAt: rawExec.createdAt || rawExec.CreatedAt || '',
            updatedAt: rawExec.updatedAt || rawExec.UpdatedAt || ''
          },
          tokens: rawTokens.map((t: any) => {
            const tokenStatus = (t.status || t.Status || 'active').toLowerCase();
            const execStatus = (rawExec.status || rawExec.Status || '').toLowerCase();
            return {
              id: t.id || t.ID || '',
              workflowExecutionId: t.workflowExecutionId || t.WorkflowExecutionID || '',
              workflowDefinitionId: t.workflowDefinitionId || t.WorkflowDefinitionID || '',
              vnamespace: t.vnamespace || t.VNamespace || '',
              currentNodeId: t.currentNodeId || t.CurrentNodeID || '',
              status: (execStatus === 'completed' && tokenStatus === 'waiting') ? 'completed' : tokenStatus,
              parentTokenId: t.parentTokenId || t.ParentTokenID || '',
              createdAt: t.createdAt || t.CreatedAt || '',
              updatedAt: t.updatedAt || t.UpdatedAt || ''
            };
          }),
          jobs: rawJobs.map((j: any) => ({
            id: j.id || j.ID || '',
            workflowExecutionId: j.workflowExecutionId || j.WorkflowExecutionID || '',
            executionTokenId: j.executionTokenId || j.ExecutionTokenID || '',
            workflowDefinitionId: j.workflowDefinitionId || j.WorkflowDefinitionID || '',
            vnamespace: j.vnamespace || j.VNamespace || '',
            activityId: j.activityId || j.ActivityID || '',
            activityName: j.activityName || j.ActivityName || '',
            activityType: j.activityType || j.ActivityType || '',
            status: (j.status || j.Status || 'pending').toLowerCase(),
            input: j.input || j.Input || null,
            output: j.output || j.Output || null,
            error: j.error || j.Error || '',
            assignedWorkerId: j.assignedWorkerId || j.AssignedWorkerID || '',
            retries: j.retries || j.Retries || 0,
            maxRetries: j.maxRetries || j.MaxRetries || 3,
            timeoutSeconds: j.timeoutSeconds || j.TimeoutSeconds || 300,
            createdAt: j.createdAt || j.CreatedAt || '',
            updatedAt: j.updatedAt || j.UpdatedAt || ''
          })),
          waitingEvents: (detailObj?.waitingEvents || detailObj?.WaitingEvents || []).map((evt: any) => ({
            id: evt.id || evt.ID || '',
            workflowDefinitionId: evt.workflowDefinitionId || evt.WorkflowDefinitionID || '',
            workflowExecutionId: evt.workflowExecutionId || evt.WorkflowExecutionID || '',
            executionTokenId: evt.executionTokenId || evt.ExecutionTokenID || '',
            eventId: evt.eventId || evt.EventID || '',
            type: evt.type || evt.Type || 'USER_INPUT',
            expectedInput: evt.expectedInput || evt.ExpectedInput || null,
            createdAt: evt.createdAt || evt.CreatedAt || '',
            updatedAt: evt.updatedAt || evt.UpdatedAt || ''
          }))
        };
        this.waitingEvents = this.selectedExecutionDetail.waitingEvents || [];
        this.initWaitEventForms();

        this.selectedExecutionUserName = this.selectedExecutionDetail.execution.userName || this.selectedExecutionDetail.execution.userId || '';
        if (this.selectedExecutionDetail.execution.userId) {
          this.usersService.resolveUser(this.selectedExecutionDetail.execution.userId).subscribe(name => {
            this.selectedExecutionUserName = name;
          });
        }

        this.loadingExecutionDetail = false;

        if (this.activeDetailTab === 'diagram') {
          setTimeout(() => this.highlightExecutionDiagram(), 250);
        }
      },
      error: (err) => {
        this.executionDetailError = ErrorUtil.formatErrorMessage(err);
        this.loadingExecutionDetail = false;
      }
    });
  }

  closeExecutionDetail(): void {
    this.showExecutionDetailModal = false;
    this.selectedExecutionDetail = null;
    this.selectedExecutionId = '';
    this.waitingEvents = [];
    this.eventFormData = {};
    this.eventRawJson = {};
    this.eventFormMode = {};
    this.submittingWaitEvent = {};
    this.waitEventErrors = {};
    this.waitEventSuccess = {};
    this.fieldValidationErrors = {};
  }

  refreshDiagram(): void {
    if (this.selectedExecutionId) {
      this.openExecutionDetail(this.selectedExecutionId);
    }
  }

  getExecutionStatusBadgeColor(status: string): string {
    switch (status?.toLowerCase()) {
      case 'completed': return 'success';
      case 'running': return 'info';
      case 'pending': return 'primary';
      case 'failed': return 'danger';
      case 'terminated': return 'danger';
      case 'cancelled': return 'warning';
      default: return 'secondary';
    }
  }

  formatJson(data: any): string {
    if (!data) return '{}';
    if (typeof data === 'string') {
      try {
        return JSON.stringify(JSON.parse(data), null, 2);
      } catch {
        return data;
      }
    }
    return JSON.stringify(data, null, 2);
  }

  selectDetailTab(tab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'waitingEvents' | 'diagram'): void {
    this.activeDetailTab = tab;
    if (tab === 'diagram') {
      setTimeout(() => this.highlightExecutionDiagram(), 250);
    }
  }

  initWaitEventForms(): void {
    for (const evt of this.waitingEvents) {
      if (!this.eventFormData[evt.id]) {
        this.eventFormData[evt.id] = {};
      }
      if (!this.fieldValidationErrors[evt.id]) {
        this.fieldValidationErrors[evt.id] = {};
      }

      const fields = this.getEventFields(evt);
      if (fields && fields.length > 0) {
        for (const f of fields) {
          if (this.eventFormData[evt.id][f.id] === undefined) {
            if (f.defaultValue !== undefined && f.defaultValue !== null && f.defaultValue !== '') {
              if (f.type === 'boolean' || f.type === 'bool') {
                this.eventFormData[evt.id][f.id] = f.defaultValue === 'true' || f.defaultValue === true;
              } else if (f.type === 'long' || f.type === 'int' || f.type === 'integer' || f.type === 'double' || f.type === 'number') {
                const num = Number(f.defaultValue);
                this.eventFormData[evt.id][f.id] = isNaN(num) ? f.defaultValue : num;
              } else {
                this.eventFormData[evt.id][f.id] = f.defaultValue;
              }
            } else if (f.type === 'boolean' || f.type === 'bool') {
              this.eventFormData[evt.id][f.id] = false;
            } else if (f.type === 'enum' && f.values && f.values.length > 0) {
              this.eventFormData[evt.id][f.id] = f.values[0].id;
            } else {
              this.eventFormData[evt.id][f.id] = '';
            }
          }
        }
      }

      // Pre-populate JSON combined from form fields + extension properties (with nested objects for dot-notation keys)
      this.eventRawJson[evt.id] = this.generateCombinedJsonPayload(evt);

      // Business Rule:
      // If at least one extension property exists, show JSON view!
      // If only a form exists (form fields exist and NO extension properties), show Form view with validations!
      // If neither, show JSON view.
      if (this.hasExtensionProperties(evt)) {
        this.eventFormMode[evt.id] = 'json';
      } else if (this.hasFormFields(evt)) {
        this.eventFormMode[evt.id] = 'form';
      } else {
        this.eventFormMode[evt.id] = 'json';
      }
    }
  }

  getEventFields(evt: WaitingEvent): WaitingEventFormField[] {
    if (!evt || !evt.expectedInput) return [];
    const fields = evt.expectedInput.fields || evt.expectedInput.Fields;
    return Array.isArray(fields) ? fields : [];
  }

  hasFormFields(evt: WaitingEvent): boolean {
    return this.getEventFields(evt).length > 0;
  }

  hasExtensionProperties(evt: WaitingEvent): boolean {
    if (!evt || !evt.expectedInput) return false;
    const exp = evt.expectedInput;
    if (exp.hasExtensionProperties === true) return true;
    const ext = this.getExtensionProperties(evt);
    return Object.keys(ext).length > 0;
  }

  getExtensionProperties(evt: WaitingEvent): Record<string, string> {
    if (!evt || !evt.expectedInput) return {};
    const exp = evt.expectedInput;
    const res: Record<string, string> = {};

    if (exp.extensionProperties && typeof exp.extensionProperties === 'object') {
      for (const [k, v] of Object.entries(exp.extensionProperties)) {
        res[k] = typeof v === 'string' ? v : String(v || '');
      }
    }

    const reserved = new Set([
      'nodeId', 'nodeName', 'nodeType', 'fields', 'schema', 'type',
      'activityId', 'activityName', 'expectedKeys', 'messageRef',
      'modelerTemplate', 'scriptFormat', 'resultVariable', 'script',
      'taskType', 'timeDuration', 'timeDate', 'timeCycle',
      'extensionProperties', 'hasFormFields', 'hasExtensionProperties'
    ]);

    for (const [k, v] of Object.entries(exp)) {
      if (!reserved.has(k) && typeof v !== 'object') {
        if (!(k in res)) {
          res[k] = typeof v === 'string' ? v : String(v || '');
        }
      }
    }

    return res;
  }

  setNestedProperty(obj: Record<string, any>, path: string, value: any): void {
    const parts = path.split('.');
    let current = obj;
    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i];
      if (!current[part] || typeof current[part] !== 'object' || Array.isArray(current[part])) {
        current[part] = {};
      }
      current = current[part];
    }
    current[parts[parts.length - 1]] = value;
  }

  generateCombinedJsonPayload(evt: WaitingEvent): string {
    const combined: Record<string, any> = {};

    // 1. Add form fields
    const fields = this.getEventFields(evt);
    for (const f of fields) {
      let defaultVal: any = '';
      if (f.defaultValue !== undefined && f.defaultValue !== null && f.defaultValue !== '') {
        if (f.type === 'boolean' || f.type === 'bool') {
          defaultVal = f.defaultValue === 'true' || f.defaultValue === true;
        } else if (f.type === 'long' || f.type === 'int' || f.type === 'integer' || f.type === 'double' || f.type === 'number') {
          const num = Number(f.defaultValue);
          defaultVal = isNaN(num) ? f.defaultValue : num;
        } else {
          defaultVal = f.defaultValue;
        }
      } else if (f.type === 'boolean' || f.type === 'bool') {
        defaultVal = false;
      } else if (f.type === 'long' || f.type === 'int' || f.type === 'integer' || f.type === 'double' || f.type === 'number') {
        defaultVal = 0;
      }

      if (this.eventFormData[evt.id] && this.eventFormData[evt.id][f.id] !== undefined) {
        defaultVal = this.eventFormData[evt.id][f.id];
      }

      this.setNestedProperty(combined, f.id, defaultVal);
    }

    // 2. Add extension properties with dot-notation support (keys only, template empty values)
    const extProps = this.getExtensionProperties(evt);
    for (const k of Object.keys(extProps)) {
      this.setNestedProperty(combined, k, '');
    }

    return JSON.stringify(combined, null, 2);
  }

  getExtensionPropertyList(evt: WaitingEvent): Array<{ key: string; value: string }> {
    const extProps = this.getExtensionProperties(evt);
    return Object.entries(extProps).map(([key, value]) => ({ key, value }));
  }

  resolveVariablePathInObject(path: string, obj: any): any {
    if (!obj || typeof obj !== 'object') return undefined;
    if (path in obj) return obj[path];
    const parts = path.split('.');
    let curr = obj;
    for (const p of parts) {
      if (curr === null || curr === undefined || typeof curr !== 'object') return undefined;
      curr = curr[p];
    }
    return curr;
  }

  setEventFormMode(evtId: string, mode: 'form' | 'json'): void {
    const evt = this.waitingEvents.find(e => e.id === evtId);
    if (evt) {
      if (mode === 'json' && this.eventFormMode[evtId] === 'form') {
        this.eventRawJson[evtId] = this.generateCombinedJsonPayload(evt);
      } else if (mode === 'form' && this.eventFormMode[evtId] === 'json') {
        try {
          const parsed = JSON.parse(this.eventRawJson[evtId] || '{}');
          const fields = this.getEventFields(evt);
          for (const f of fields) {
            const val = this.resolveVariablePathInObject(f.id, parsed);
            if (val !== undefined) {
              this.eventFormData[evtId][f.id] = val;
            }
          }
        } catch {}
      }
    }
    this.eventFormMode[evtId] = mode;
  }

  getFieldConstraint(field: WaitingEventFormField, constraintName: string): string | null {
    if (!field.constraints) return null;
    const c = field.constraints.find(item => item.name.toLowerCase() === constraintName.toLowerCase());
    return c ? (c.config || '') : null;
  }

  validateSingleField(evt: WaitingEvent, field: WaitingEventFormField): string {
    const val = this.eventFormData[evt.id]?.[field.id];
    const required = field.required || this.getFieldConstraint(field, 'required') !== null;

    if (required) {
      if (val === undefined || val === null || (typeof val === 'string' && val.trim() === '')) {
        return `Field "${field.label || field.id}" is required`;
      }
    }

    if (val !== undefined && val !== null && val !== '') {
      const typeLower = (field.type || 'string').toLowerCase();

      if (['long', 'int', 'integer', 'double', 'float', 'number'].includes(typeLower)) {
        const num = Number(val);
        if (isNaN(num)) {
          return `Field "${field.label || field.id}" must be a valid number`;
        }
        const minStr = this.getFieldConstraint(field, 'min');
        if (minStr !== null && minStr !== '') {
          const minVal = parseFloat(minStr);
          if (!isNaN(minVal) && num < minVal) {
            return `Value must be at least ${minVal}`;
          }
        }
        const maxStr = this.getFieldConstraint(field, 'max');
        if (maxStr !== null && maxStr !== '') {
          const maxVal = parseFloat(maxStr);
          if (!isNaN(maxVal) && num > maxVal) {
            return `Value cannot exceed ${maxVal}`;
          }
        }
      }

      if (typeof val === 'string') {
        const minLenStr = this.getFieldConstraint(field, 'minlength');
        if (minLenStr !== null && minLenStr !== '') {
          const minLen = parseInt(minLenStr, 10);
          if (!isNaN(minLen) && val.length < minLen) {
            return `Minimum length is ${minLen} characters`;
          }
        }
        const maxLenStr = this.getFieldConstraint(field, 'maxlength');
        if (maxLenStr !== null && maxLenStr !== '') {
          const maxLen = parseInt(maxLenStr, 10);
          if (!isNaN(maxLen) && val.length > maxLen) {
            return `Maximum length is ${maxLen} characters`;
          }
        }
        const patternStr = this.getFieldConstraint(field, 'pattern');
        if (patternStr !== null && patternStr !== '') {
          try {
            const regex = new RegExp(patternStr);
            if (!regex.test(val)) {
              return `Value does not match required pattern: ${patternStr}`;
            }
          } catch {}
        }
      }
    }

    return '';
  }

  validateField(evt: WaitingEvent, field: WaitingEventFormField): void {
    if (!this.fieldValidationErrors[evt.id]) {
      this.fieldValidationErrors[evt.id] = {};
    }
    const err = this.validateSingleField(evt, field);
    if (err) {
      this.fieldValidationErrors[evt.id][field.id] = err;
    } else {
      delete this.fieldValidationErrors[evt.id][field.id];
    }
  }

  validateAllFields(evt: WaitingEvent): boolean {
    if (!this.fieldValidationErrors[evt.id]) {
      this.fieldValidationErrors[evt.id] = {};
    }
    const fields = this.getEventFields(evt);
    let hasErrors = false;
    for (const f of fields) {
      const err = this.validateSingleField(evt, f);
      if (err) {
        this.fieldValidationErrors[evt.id][f.id] = err;
        hasErrors = true;
      } else {
        delete this.fieldValidationErrors[evt.id][f.id];
      }
    }
    return !hasErrors;
  }

  submitWaitEvent(evt: WaitingEvent): void {
    this.submittingWaitEvent[evt.id] = true;
    this.waitEventErrors[evt.id] = '';
    this.waitEventSuccess[evt.id] = '';

    let payload: any = {};
    const mode = this.eventFormMode[evt.id] || (this.hasExtensionProperties(evt) ? 'json' : (this.hasFormFields(evt) ? 'form' : 'json'));

    if (mode === 'json') {
      try {
        payload = JSON.parse(this.eventRawJson[evt.id] || '{}');
      } catch (e: any) {
        this.waitEventErrors[evt.id] = 'Invalid JSON: ' + (e?.message || 'Syntax error');
        this.submittingWaitEvent[evt.id] = false;
        return;
      }
    } else {
      const isValid = this.validateAllFields(evt);
      if (!isValid) {
        this.waitEventErrors[evt.id] = 'Please fix the validation errors in the form before submitting.';
        this.submittingWaitEvent[evt.id] = false;
        return;
      }

      const formData = this.eventFormData[evt.id] || {};
      const fields = this.getEventFields(evt);
      for (const f of fields) {
        let val = formData[f.id];
        if (f.type === 'long' || f.type === 'int' || f.type === 'integer') {
          if (val !== undefined && val !== null && val !== '') {
            val = parseInt(val, 10);
          }
        } else if (f.type === 'double' || f.type === 'float' || f.type === 'number') {
          if (val !== undefined && val !== null && val !== '') {
            val = parseFloat(val);
          }
        } else if (f.type === 'boolean' || f.type === 'bool') {
          val = !!val;
        }
        this.setNestedProperty(payload, f.id, val);
      }
    }

    const req$ = this.scope === 'global'
      ? this.workflowsService.completeGlobalWaitEvent(evt.id, payload)
      : this.workflowsService.completeTenantWaitEvent(this.tenantCode, evt.id, payload);

    req$.subscribe({
      next: () => {
        this.submittingWaitEvent[evt.id] = false;
        this.waitEventSuccess[evt.id] = 'Waiting event resumed successfully!';
        setTimeout(() => {
          if (this.selectedExecutionId) {
            this.openExecutionDetail(this.selectedExecutionId);
          }
          this.loadExecutions();
        }, 600);
      },
      error: (err) => {
        this.submittingWaitEvent[evt.id] = false;
        this.waitEventErrors[evt.id] = ErrorUtil.formatErrorMessage(err);
      }
    });
  }

  private decodePayload(payloadRaw: any): string {
    if (!payloadRaw) return '';
    let decoded = '';
    if (typeof payloadRaw === 'string') {
      try {
        decoded = atob(payloadRaw);
      } catch {
        decoded = payloadRaw;
      }
    } else {
      decoded = JSON.stringify(payloadRaw, null, 2);
    }
    decoded = decoded.trim();
    if (decoded.startsWith('"') && decoded.endsWith('"')) {
      try {
        const unescaped = JSON.parse(decoded);
        if (typeof unescaped === 'string') {
          decoded = unescaped.trim();
        }
      } catch {}
    }
    return decoded;
  }

  getExecutionSnapshotPayload(): string {
    if (!this.selectedExecutionDetail?.execution) return DEFAULT_BPMN_XML;
    const snap = this.selectedExecutionDetail.execution.payloadSnapshot;
    if (snap) {
      return this.decodePayload(snap) || DEFAULT_BPMN_XML;
    }
    if (this.workflow?.payload) {
      return this.decodePayload(this.workflow.payload) || DEFAULT_BPMN_XML;
    }
    return DEFAULT_BPMN_XML;
  }

  highlightExecutionDiagram(): void {
    if (!this.executionBpmnDesigner || !this.selectedExecutionDetail) return;
    this.executionBpmnDesigner.clearHighlights();

    const execStatus = (this.selectedExecutionDetail.execution?.status || '').toLowerCase();
    const markers: Array<{ id: string; type: 'active' | 'error' | 'waiting' | 'completed' }> = [];

    if (execStatus === 'completed') {
      // Strictly highlight End Event node(s) in green when workflow execution has completed
      if (this.executionBpmnDesigner) {
        const endEventIds = this.executionBpmnDesigner.getEndEventIds();
        endEventIds.forEach(id => markers.push({ id, type: 'completed' }));
      }
    } else if (execStatus === 'running' || execStatus === 'pending') {
      // Highlight active/waiting token positions if execution is currently in progress
      const activeTokens = this.selectedExecutionDetail.tokens || [];
      activeTokens.forEach(t => {
        const statusLower = (t.status || '').toLowerCase();
        if (t.currentNodeId) {
          if (statusLower === 'waiting') {
            markers.push({ id: t.currentNodeId, type: 'waiting' });
          } else if (statusLower === 'active' || statusLower === 'running' || statusLower === 'pending') {
            markers.push({ id: t.currentNodeId, type: 'active' });
          }
        }
      });
    } else if (execStatus === 'failed' || execStatus === 'terminated') {
      // Highlight failed/cancelled token positions if workflow execution failed
      const tokens = this.selectedExecutionDetail.tokens || [];
      tokens.forEach(t => {
        if (t.currentNodeId && (t.status === 'cancelled' || t.status === 'failed')) {
          markers.push({ id: t.currentNodeId, type: 'error' });
        }
      });
    }

    const jobs = this.selectedExecutionDetail.jobs || [];
    jobs.forEach(j => {
      if (j.activityId && (j.status === 'failed' || j.error)) {
        markers.push({ id: j.activityId, type: 'error' });
      }
    });

    this.executionBpmnDesigner.highlightElements(markers);
  }

  navigateBack(): void {
    if (this.tenantCode) {
      this.router.navigate(['/tenants', this.tenantCode, 'management'], { queryParams: { tab: 'workflows' } });
    } else {
      this.router.navigate(['/workflows']);
    }
  }
}
