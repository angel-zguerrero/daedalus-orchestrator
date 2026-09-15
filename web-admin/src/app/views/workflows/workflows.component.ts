import { Component, OnInit, Input, OnChanges, SimpleChanges, ViewChild } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { map, switchMap, startWith, debounceTime } from 'rxjs/operators';
import {
  TableModule,
  UtilitiesModule,
  ButtonModule,
  ModalModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  NavModule,
  TabsModule,
  AccordionComponent,
  AccordionItemComponent,
  TemplateIdDirective,
  AccordionButtonDirective
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition, WorkflowExecution, WorkflowExecutionDetail } from './services/workflows.service';
import { VNamespacesService } from '../tenants/tenant-management/services/vnamespaces.service';
import { ErrorUtil } from '../../shared/utils/error.util';
import { QueueDetailComponent } from '../tenants/tenant-management/queues/queue-detail/queue-detail.component';
import { BpmnDesignerComponent, DEFAULT_BPMN_XML } from '../../shared/components/bpmn-designer/bpmn-designer.component';

@Component({
  selector: 'app-workflows',
  templateUrl: './workflows.component.html',
  styleUrls: ['./workflows.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    AsyncPipe,
    FormsModule,
    ReactiveFormsModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    NavModule,
    TabsModule,
    AccordionComponent,
    AccordionItemComponent,
    TemplateIdDirective,
    AccordionButtonDirective,
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    QueueDetailComponent,
    BpmnDesignerComponent
  ]
})
export class WorkflowsComponent implements OnInit, OnChanges {
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';

  workflows: WorkflowDefinition[] = [];
  filteredWorkflows: WorkflowDefinition[] = [];
  loading: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  searchQuery: string = '';
  selectedStatusFilter: 'all' | 'active' | 'inactive' = 'all';

  // Pagination
  cursor: string = '';
  nextCursor: string = '';
  cursors: string[] = [''];
  pageSize: number = 20;

  // Create / Edit Modal
  showModal: boolean = false;
  showAdvancedSettings: boolean = false;
  showUnsavedConfirmModal: boolean = false;
  hasUnsavedChanges: boolean = false;
  workflowForm: FormGroup;
  isEditing: boolean = false;
  editingWorkflowId: string = '';
  @ViewChild('bpmnDesigner') bpmnDesigner?: BpmnDesignerComponent;

  toggleAdvancedSettings(): void {
    this.showAdvancedSettings = !this.showAdvancedSettings;
  }

  // Detail / Payload Modal
  showDetailModal: boolean = false;
  selectedWorkflow: WorkflowDefinition | null = null;
  payloadDisplayText: string = '';

  // Delete Confirm Modal
  showDeleteModal: boolean = false;
  workflowToDelete: WorkflowDefinition | null = null;

  // Execute Workflow Modal
  showExecuteModal: boolean = false;
  executingWorkflow: WorkflowDefinition | null = null;
  executeInputJson: string = '{\n  "approved": true\n}';
  executionKey: string = '';
  executing: boolean = false;
  executionResult: any = null;
  executeErrorMessage: string = '';

  // Executions List Modal
  showExecutionsModal: boolean = false;
  executionsWorkflow: WorkflowDefinition | null = null;
  executionsList: WorkflowExecution[] = [];
  loadingExecutions: boolean = false;
  executionsErrorMessage: string = '';
  executionStatusFilter: string = '';

  // Execution Detail Modal
  showExecutionDetailModal: boolean = false;
  selectedExecutionDetail: WorkflowExecutionDetail | null = null;
  loadingExecutionDetail: boolean = false;
  executionDetailError: string = '';
  activeDetailTab: 'overview' | 'payloads' | 'activities' | 'tokens' = 'overview';

  // VNamespace properties
  vnamespaceCtrl = new FormControl('default');
  filteredVNamespaces!: Observable<any[]>;
  loadingVNamespaces: boolean = false;

  vnamespaceFilterCtrl = new FormControl('');
  filteredFilterVNamespaces!: Observable<any[]>;
  selectedVNamespaceFilter: string = '';

  constructor(
    private fb: FormBuilder,
    private workflowsService: WorkflowsService,
    private vNamespacesService: VNamespacesService
  ) {
    this.workflowForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      vnamespace: this.vnamespaceCtrl,
      description: [''],
      version: [1, [Validators.required, Validators.min(1)]],
      payloadFormat: ['json', Validators.required],
      payload: ['{}'],
      maxDurationSeconds: [3600, [Validators.required, Validators.min(0)]],
      isActive: [true]
    });

    this.workflowForm.valueChanges.subscribe(() => {
      if (this.showModal) {
        this.hasUnsavedChanges = true;
      }
    });

    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );

    this.filteredFilterVNamespaces = this.vnamespaceFilterCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );
  }

  ngOnInit(): void {
    this.loadWorkflows();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['tenantCode'] || changes['scope']) {
      this.cursor = '';
      this.cursors = [''];
      this.loadWorkflows();
    }
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
      } catch {
        // ignore parse error
      }
    }
    return decoded;
  }

  private normalizeWorkflow(w: any): WorkflowDefinition {
    return {
      id: w.id || w.ID || '',
      code: w.code || w.Code || '',
      vnamespace: w.vnamespace || w.VNamespace || '',
      name: w.name || w.Name || w.code || w.Code || '',
      description: w.description || w.Description || '',
      version: w.version || w.Version || 1,
      payload: w.payload || w.Payload || '',
      payloadFormat: (w.payloadFormat || w.PayloadFormat || 'json').toLowerCase() as 'json' | 'yaml' | 'bpmn',
      maxDurationSeconds: w.maxDurationSeconds || w.MaxDurationSeconds || 0,
      isActive: w.isActive !== undefined ? w.isActive : (w.IsActive !== undefined ? w.IsActive : true),
      scope: (w.scope || w.Scope || this.scope).toLowerCase() as 'global' | 'tenant',
      tenantId: w.tenantId || w.TenantID || '',
      createdAt: w.createdAt || w.CreatedAt || '',
      updatedAt: w.updatedAt || w.UpdatedAt || ''
    };
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    if (!this.tenantCode) {
      return of([]);
    }
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantCode, '', 20, value).pipe(
      map(response => {
        this.loadingVNamespaces = false;
        return response.data || response.result?.Entities || response.entities || [];
      })
    );
  }

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = value;
    this.cursor = '';
    this.cursors = [''];
    this.loadWorkflows();
  }

  loadWorkflows(): void {
    this.loading = true;
    this.showAlert = false;
    this.errorMessage = '';

    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflows(this.pageSize, this.cursor, this.selectedVNamespaceFilter)
      : this.workflowsService.getTenantWorkflows(this.tenantCode, this.pageSize, this.cursor, this.selectedVNamespaceFilter);

    req$.subscribe({
      next: (res) => {
        const rawEntities = res.Entities || res.entities || [];
        this.workflows = rawEntities.map((w: any) => this.normalizeWorkflow(w));
        this.nextCursor = res.Cursor || res.cursor || '';
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.loading = false;
      }
    });
  }

  applyFilters(): void {
    let result = [...this.workflows];

    if (this.selectedStatusFilter === 'active') {
      result = result.filter(w => w.isActive);
    } else if (this.selectedStatusFilter === 'inactive') {
      result = result.filter(w => !w.isActive);
    }

    if (this.searchQuery.trim()) {
      const q = this.searchQuery.toLowerCase().trim();
      result = result.filter(w =>
        w.code.toLowerCase().includes(q) ||
        (w.name && w.name.toLowerCase().includes(q)) ||
        (w.description && w.description.toLowerCase().includes(q))
      );
    }

    this.filteredWorkflows = result;
  }

  searchWorkflows(): void {
    this.applyFilters();
  }

  nextPage(): void {
    if (this.nextCursor) {
      this.cursors.push(this.nextCursor);
      this.cursor = this.nextCursor;
      this.loadWorkflows();
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.cursor = this.cursors[this.cursors.length - 1];
      this.loadWorkflows();
    }
  }

  openCreateModal(): void {
    this.isEditing = false;
    this.editingWorkflowId = '';
    this.showAdvancedSettings = false;
    this.workflowForm.reset({
      vnamespace: this.scope === 'global' ? '' : 'default',
      version: 1,
      payloadFormat: 'bpmn',
      payload: DEFAULT_BPMN_XML,
      maxDurationSeconds: 3600,
      isActive: true
    });
    this.workflowForm.get('code')?.enable();
    this.hasUnsavedChanges = false;
    this.showModal = true;
    setTimeout(() => {
      if (this.bpmnDesigner) {
        this.bpmnDesigner.refresh();
      }
    }, 150);
  }

  openEditModal(wf: WorkflowDefinition): void {
    this.isEditing = true;
    this.editingWorkflowId = wf.id || '';
    this.showAdvancedSettings = false;
    const decodedPayload = this.decodePayload(wf.payload);

    this.workflowForm.patchValue({
      code: wf.code,
      name: wf.name,
      vnamespace: this.scope === 'global' ? '' : (wf.vnamespace || 'default'),
      description: wf.description,
      version: wf.version,
      payloadFormat: 'bpmn',
      payload: decodedPayload || DEFAULT_BPMN_XML,
      maxDurationSeconds: wf.maxDurationSeconds,
      isActive: wf.isActive
    });
    this.workflowForm.get('code')?.disable();
    this.hasUnsavedChanges = false;
    this.showModal = true;
    setTimeout(() => {
      if (this.bpmnDesigner) {
        this.bpmnDesigner.refresh();
      }
    }, 150);
  }

  onBpmnXmlChange(xml: string): void {
    this.workflowForm.patchValue({ payload: xml }, { emitEvent: false });
    if (this.showModal) {
      this.hasUnsavedChanges = true;
    }
  }

  requestCloseModal(): void {
    if (this.hasUnsavedChanges) {
      this.showUnsavedConfirmModal = true;
    } else {
      this.showModal = false;
      this.hasUnsavedChanges = false;
    }
  }

  onModalVisibleChange(visible: boolean): void {
    if (!visible) {
      if (this.hasUnsavedChanges) {
        setTimeout(() => {
          this.showModal = true;
          this.showUnsavedConfirmModal = true;
        }, 10);
      } else {
        this.showModal = false;
      }
    }
  }

  cancelCloseUnsavedModal(): void {
    this.showUnsavedConfirmModal = false;
  }

  confirmDiscardAndClose(): void {
    this.hasUnsavedChanges = false;
    this.showUnsavedConfirmModal = false;
    this.showModal = false;
  }

  async saveWorkflowAndClose(): Promise<void> {
    await this.saveWorkflow(true);
  }

  // Workflow Queues
  workflowQueues: any[] = [];
  loadingWorkflowQueues: boolean = false;

  openDetailModal(wf: WorkflowDefinition): void {
    this.selectedWorkflow = wf;
    this.payloadDisplayText = this.decodePayload(wf.payload);
    this.showDetailModal = true;
    this.loadWorkflowQueues(wf);
  }

  loadWorkflowQueues(wf: WorkflowDefinition): void {
    if (!wf.id) return;
    this.loadingWorkflowQueues = true;
    const queues$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflowQueues(wf.id)
      : this.workflowsService.getTenantWorkflowQueues(this.tenantCode, wf.id);

    queues$.subscribe({
      next: (res) => {
        this.workflowQueues = res.queues || [];
        this.loadingWorkflowQueues = false;
      },
      error: () => {
        this.workflowQueues = [];
        this.loadingWorkflowQueues = false;
      }
    });
  }

  async saveWorkflow(closeAfterSave: boolean = true): Promise<void> {
    if (this.workflowForm.invalid) return;

    if (this.bpmnDesigner) {
      const xml = await this.bpmnDesigner.getXml();
      if (xml) {
        this.workflowForm.patchValue({ payload: xml }, { emitEvent: false });
      }
    }

    const val = this.workflowForm.getRawValue();
    const payload: Partial<WorkflowDefinition> = {
      code: val.code ? val.code.trim() : '',
      name: val.name ? val.name.trim() : '',
      vnamespace: this.scope === 'global' ? '' : (val.vnamespace ? val.vnamespace.trim() : 'default'),
      description: val.description,
      version: Number(val.version),
      payloadFormat: 'bpmn',
      payload: val.payload,
      maxDurationSeconds: Number(val.maxDurationSeconds),
      isActive: Boolean(val.isActive),
      scope: this.scope
    };

    if (this.isEditing) {
      const update$ = this.scope === 'global'
        ? this.workflowsService.updateGlobalWorkflow(this.editingWorkflowId, payload)
        : this.workflowsService.updateTenantWorkflow(this.tenantCode, this.editingWorkflowId, payload);

      update$.subscribe({
        next: () => {
          this.hasUnsavedChanges = false;
          this.showUnsavedConfirmModal = false;
          if (closeAfterSave) {
            this.showModal = false;
          }
          this.successMsg = 'Workflow definition updated successfully.';
          this.loadWorkflows();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    } else {
      const create$ = this.scope === 'global'
        ? this.workflowsService.createGlobalWorkflow(payload)
        : this.workflowsService.createTenantWorkflow(this.tenantCode, payload);

      create$.subscribe({
        next: (res: any) => {
          this.hasUnsavedChanges = false;
          this.showUnsavedConfirmModal = false;
          const createdEntity = res?.Entity || res?.entity || res;
          if (createdEntity && createdEntity.id) {
            this.isEditing = true;
            this.editingWorkflowId = createdEntity.id;
            this.workflowForm.get('code')?.disable();
          }
          if (closeAfterSave) {
            this.showModal = false;
          }
          this.successMsg = 'Workflow definition created successfully.';
          this.loadWorkflows();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    }
  }

  confirmDelete(wf: WorkflowDefinition): void {
    this.workflowToDelete = wf;
    this.showDeleteModal = true;
  }

  deleteWorkflow(): void {
    if (!this.workflowToDelete || !this.workflowToDelete.id) return;

    const del$ = this.scope === 'global'
      ? this.workflowsService.deleteGlobalWorkflow(this.workflowToDelete.id)
      : this.workflowsService.deleteTenantWorkflow(this.tenantCode, this.workflowToDelete.id);

    del$.subscribe({
      next: () => {
        this.showDeleteModal = false;
        this.workflowToDelete = null;
        this.successMsg = 'Workflow definition deleted successfully.';
        this.loadWorkflows();
        setTimeout(() => this.successMsg = '', 3000);
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
      }
    });
  }

  // --- EXECUTE WORKFLOW HANDLERS ---
  openExecuteModal(wf: WorkflowDefinition): void {
    this.executingWorkflow = wf;
    this.executeInputJson = '{\n  "approved": true\n}';
    this.executionKey = '';
    this.executionResult = null;
    this.executeErrorMessage = '';
    this.showExecuteModal = true;
  }

  openExecuteModalFromForm(): void {
    if (this.selectedWorkflow) {
      this.openExecuteModal(this.selectedWorkflow);
    } else if (this.isEditing && this.editingWorkflowId) {
      const currentWf: WorkflowDefinition = {
        id: this.editingWorkflowId,
        code: this.workflowForm.get('code')?.value,
        name: this.workflowForm.get('name')?.value,
        vnamespace: this.workflowForm.get('vnamespace')?.value,
        version: this.workflowForm.get('version')?.value || 1,
        payloadFormat: 'json',
        maxDurationSeconds: 3600,
        isActive: true,
        scope: this.scope
      };
      this.openExecuteModal(currentWf);
    }
  }

  confirmExecuteWorkflow(): void {
    if (!this.executingWorkflow || !this.executingWorkflow.id) return;

    let inputObj = {};
    if (this.executeInputJson && this.executeInputJson.trim()) {
      try {
        inputObj = JSON.parse(this.executeInputJson);
      } catch (e: any) {
        this.executeErrorMessage = 'Invalid JSON input payload: ' + e.message;
        return;
      }
    }

    this.executing = true;
    this.executeErrorMessage = '';
    this.executionResult = null;

    const vns = this.executingWorkflow.vnamespace || 'default';
    const exec$ = this.scope === 'global'
      ? this.workflowsService.executeGlobalWorkflow(this.executingWorkflow.id, inputObj, this.executionKey, vns)
      : this.workflowsService.executeTenantWorkflow(this.tenantCode, this.executingWorkflow.id, inputObj, this.executionKey, vns);

    exec$.subscribe({
      next: (res: any) => {
        this.executing = false;
        this.executionResult = res?.Result || res?.result || res;
        this.successMsg = `Workflow execution started successfully! Execution ID: ${this.executionResult?.id || ''}`;
        setTimeout(() => this.successMsg = '', 6000);
      },
      error: (err) => {
        this.executing = false;
        this.executeErrorMessage = ErrorUtil.formatErrorMessage(err);
      }
    });
  }

  closeExecuteModal(): void {
    this.showExecuteModal = false;
    this.executingWorkflow = null;
    this.executionResult = null;
  }

  // --- EXECUTIONS LIST HANDLERS ---
  openExecutionsModal(wf: WorkflowDefinition): void {
    this.executionsWorkflow = wf;
    this.executionStatusFilter = '';
    this.showExecutionsModal = true;
    this.loadExecutions();
  }

  loadExecutions(): void {
    if (!this.executionsWorkflow || !this.executionsWorkflow.id) return;
    this.loadingExecutions = true;
    this.executionsErrorMessage = '';

    const vns = this.executionsWorkflow.vnamespace || '';
    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalExecutions(this.executionsWorkflow.id, this.executionStatusFilter, 50, '', vns)
      : this.workflowsService.getTenantExecutions(this.tenantCode, this.executionsWorkflow.id, this.executionStatusFilter, 50, '', vns);

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
          startedAt: e.startedAt || e.StartedAt || null,
          completedAt: e.completedAt || e.CompletedAt || null,
          createdAt: e.createdAt || e.CreatedAt || '',
          updatedAt: e.updatedAt || e.UpdatedAt || ''
        }));
        this.loadingExecutions = false;
      },
      error: (err) => {
        this.executionsErrorMessage = ErrorUtil.formatErrorMessage(err);
        this.loadingExecutions = false;
      }
    });
  }

  closeExecutionsModal(): void {
    this.showExecutionsModal = false;
    this.executionsWorkflow = null;
    this.executionsList = [];
  }

  // --- EXECUTION DETAIL HANDLERS ---
  openExecutionDetail(executionId: string): void {
    this.showExecutionDetailModal = true;
    this.loadingExecutionDetail = true;
    this.executionDetailError = '';
    this.selectedExecutionDetail = null;
    this.activeDetailTab = 'overview';

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
            vnamespace: rawExec.vnamespace || rawExec.VNamespace || '',
            executionKey: rawExec.executionKey || rawExec.ExecutionKey || '',
            status: (rawExec.status || rawExec.Status || 'pending').toLowerCase(),
            input: rawExec.input || rawExec.Input || null,
            output: rawExec.output || rawExec.Output || null,
            stateData: rawExec.stateData || rawExec.StateData || null,
            error: rawExec.error || rawExec.Error || '',
            startedAt: rawExec.startedAt || rawExec.StartedAt || null,
            completedAt: rawExec.completedAt || rawExec.CompletedAt || null,
            createdAt: rawExec.createdAt || rawExec.CreatedAt || '',
            updatedAt: rawExec.updatedAt || rawExec.UpdatedAt || ''
          },
          tokens: rawTokens.map((t: any) => ({
            id: t.id || t.ID || '',
            workflowExecutionId: t.workflowExecutionId || t.WorkflowExecutionID || '',
            workflowDefinitionId: t.workflowDefinitionId || t.WorkflowDefinitionID || '',
            vnamespace: t.vnamespace || t.VNamespace || '',
            currentNodeId: t.currentNodeId || t.CurrentNodeID || '',
            status: (t.status || t.Status || 'active').toLowerCase(),
            parentTokenId: t.parentTokenId || t.ParentTokenID || '',
            createdAt: t.createdAt || t.CreatedAt || '',
            updatedAt: t.updatedAt || t.UpdatedAt || ''
          })),
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
          }))
        };
        this.loadingExecutionDetail = false;
      },
      error: (err) => {
        this.executionDetailError = ErrorUtil.formatErrorMessage(err);
        this.loadingExecutionDetail = false;
      }
    });
  }

  closeExecutionDetailModal(): void {
    this.showExecutionDetailModal = false;
    this.selectedExecutionDetail = null;
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
}
