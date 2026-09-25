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
import { WorkflowsService, WorkflowDefinition, WorkflowExecution, WorkflowExecutionDetail, WorkflowDefinitionVersion } from './services/workflows.service';
import { VNamespacesService } from '../tenants/tenant-management/services/vnamespaces.service';
import { ErrorUtil } from '../../shared/utils/error.util';
import { QueueDetailComponent } from '../tenants/tenant-management/queues/queue-detail/queue-detail.component';
import { BpmnDesignerComponent, DEFAULT_BPMN_XML } from '../../shared/components/bpmn-designer/bpmn-designer.component';
import { BpmnFormParserUtil, GeneratedFormField } from '../../shared/utils/bpmn-form-parser.util';
import { DesignErrorsConsoleComponent } from '../../shared/components/design-errors-console/design-errors-console.component';

import { Router, RouterModule } from '@angular/router';

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
    RouterModule,
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
    BpmnDesignerComponent,
    DesignErrorsConsoleComponent
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
  currentWorkflowVersion: number = 1;
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
  showExecutionSuccessModal: boolean = false;
  executingWorkflow: WorkflowDefinition | null = null;
  lastExecutedWorkflow: WorkflowDefinition | null = null;
  executeInputJson: string = '{\n  "approved": true\n}';
  executionKey: string = '';
  executionOnVersionChange: string = 'continue';
  executing: boolean = false;
  executionResult: any = null;
  executeErrorMessage: string = '';
  startFormFields: GeneratedFormField[] = [];
  formValues: { [key: string]: any } = {};
  formErrors: { [key: string]: string } = {};
  hasStartForm: boolean = false;
  executionInputMode: 'form' | 'json' = 'form';

  // Executions List Modal
  showExecutionsModal: boolean = false;
  executionsWorkflow: WorkflowDefinition | null = null;
  executionsList: WorkflowExecution[] = [];
  loadingExecutions: boolean = false;
  executionsErrorMessage: string = '';
  executionStatusFilter: string = '';

  @ViewChild('executionBpmnDesigner') executionBpmnDesigner?: BpmnDesignerComponent;

  // Execution Detail Modal
  showExecutionDetailModal: boolean = false;
  selectedExecutionDetail: WorkflowExecutionDetail | null = null;
  loadingExecutionDetail: boolean = false;
  executionDetailError: string = '';
  activeDetailTab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'diagram' = 'overview';

  // Version History Modal
  showVersionHistoryModal: boolean = false;
  selectedVersionWorkflow: WorkflowDefinition | null = null;
  versionHistoryList: WorkflowDefinitionVersion[] = [];
  loadingVersionHistory: boolean = false;
  versionHistoryError: string = '';
  showVersionDiagramModal: boolean = false;
  selectedVersionRecord: WorkflowDefinitionVersion | null = null;
  versionDiagramPayload: string = '';

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
    private vNamespacesService: VNamespacesService,
    private router: Router
  ) {
    this.workflowForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      vnamespace: this.vnamespaceCtrl,
      description: [''],
      onVersionChange: ['defined_in_execution', Validators.required],
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
      onVersionChange: w.onVersionChange || w.OnVersionChange || 'defined_in_execution',
      payload: w.payload || w.Payload || '',
      payloadFormat: (w.payloadFormat || w.PayloadFormat || 'json').toLowerCase() as 'json' | 'yaml' | 'bpmn',
      maxDurationSeconds: w.maxDurationSeconds || w.MaxDurationSeconds || 0,
      isActive: w.isActive !== undefined ? w.isActive : (w.IsActive !== undefined ? w.IsActive : true),
      scope: (w.scope || w.Scope || this.scope).toLowerCase() as 'global' | 'tenant',
      tenantId: w.tenantId || w.TenantID || '',
      hasDesignErrors: w.hasDesignErrors !== undefined ? w.hasDesignErrors : (w.HasDesignErrors !== undefined ? w.HasDesignErrors : false),
      designErrorMessages: w.designErrorMessages || w.DesignErrorMessages || [],
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

  navigateToCreate(): void {
    const queryParams = this.scope === 'tenant' ? { tenantCode: this.tenantCode } : {};
    this.router.navigate(['/workflows', 'new'], { queryParams });
  }

  navigateToEdit(wf: WorkflowDefinition): void {
    const queryParams = this.scope === 'tenant' ? { tenantCode: this.tenantCode } : {};
    this.router.navigate(['/workflows', wf.id, 'edit'], { queryParams });
  }

  navigateToExecutions(wf: WorkflowDefinition): void {
    const queryParams = this.scope === 'tenant' ? { tenantCode: this.tenantCode } : {};
    this.router.navigate(['/workflows', wf.id, 'executions'], { queryParams });
  }

  navigateToVersionHistory(wf: WorkflowDefinition): void {
    const queryParams = this.scope === 'tenant' ? { tenantCode: this.tenantCode } : {};
    this.router.navigate(['/workflows', wf.id, 'versions'], { queryParams });
  }

  openCreateModal(): void {
    this.navigateToCreate();
  }

  openEditModal(wf: WorkflowDefinition): void {
    this.navigateToEdit(wf);
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
    if (!this.workflowForm.get('name')?.value?.trim()) {
      this.workflowForm.patchValue({ name: 'Untitled Workflow' });
    }
    if (this.workflowForm.get('maxDurationSeconds')?.value === null || this.workflowForm.get('maxDurationSeconds')?.value === undefined) {
      this.workflowForm.patchValue({ maxDurationSeconds: 3600 });
    }

    let hasDesignErrors = false;
    let designErrorMessages: string[] = [];

    if (this.bpmnDesigner) {
      const xml = await this.bpmnDesigner.getXml();
      if (xml) {
        this.workflowForm.patchValue({ payload: xml }, { emitEvent: false });
      }
      const lintResult = this.bpmnDesigner.runLintValidation();
      hasDesignErrors = lintResult.hasErrors;
      designErrorMessages = lintResult.errors;
    }

    const val = this.workflowForm.getRawValue();
    const payload: Partial<WorkflowDefinition> = {
      code: val.code ? val.code.trim() : '',
      name: val.name ? val.name.trim() : 'Untitled Workflow',
      vnamespace: this.scope === 'global' ? '' : (val.vnamespace ? val.vnamespace.trim() : 'default'),
      description: val.description,
      onVersionChange: val.onVersionChange || 'defined_in_execution',
      payloadFormat: 'bpmn',
      payload: val.payload,
      maxDurationSeconds: Number(val.maxDurationSeconds),
      isActive: Boolean(val.isActive),
      scope: this.scope,
      hasDesignErrors: hasDesignErrors,
      designErrorMessages: designErrorMessages
    };

    if (this.isEditing) {
      const update$ = this.scope === 'global'
        ? this.workflowsService.updateGlobalWorkflow(this.editingWorkflowId, payload)
        : this.workflowsService.updateTenantWorkflow(this.tenantCode, this.editingWorkflowId, payload);

      update$.subscribe({
        next: (res: any) => {
          this.hasUnsavedChanges = false;
          this.showUnsavedConfirmModal = false;
          const updatedEntity = res?.Entity || res?.entity || res;
          if (updatedEntity && updatedEntity.version) {
            this.currentWorkflowVersion = updatedEntity.version;
          }
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
            if (createdEntity.version) {
              this.currentWorkflowVersion = createdEntity.version;
            }
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

  get isExecutionPolicyDisabled(): boolean {
    return !!(this.executingWorkflow?.onVersionChange && this.executingWorkflow.onVersionChange !== 'defined_in_execution');
  }

  // --- EXECUTE WORKFLOW HANDLERS ---
  async openExecuteModal(wf: WorkflowDefinition): Promise<void> {
    this.executingWorkflow = wf;
    this.executionKey = '';
    this.executionResult = null;
    this.executeErrorMessage = '';

    const wfPolicy = wf.onVersionChange || (wf as any).OnVersionChange || 'defined_in_execution';
    if (wfPolicy === 'continue' || wfPolicy === 'restart') {
      this.executionOnVersionChange = wfPolicy;
    } else {
      this.executionOnVersionChange = 'continue';
    }

    // Extract BPMN XML payload from definition or active editor
    let xmlPayload = '';
    if (this.isEditing && this.bpmnDesigner) {
      xmlPayload = await this.bpmnDesigner.getXml();
    } else if (wf && wf.payload) {
      xmlPayload = this.decodePayload(wf.payload);
    } else if (this.workflowForm && this.workflowForm.get('payload')?.value) {
      xmlPayload = this.decodePayload(this.workflowForm.get('payload')?.value);
    }

    // Extract Generated Task Form fields from StartEvent
    this.startFormFields = BpmnFormParserUtil.extractStartFormFields(xmlPayload);
    this.formValues = {};
    this.formErrors = {};

    if (this.startFormFields.length > 0) {
      this.hasStartForm = true;
      this.executionInputMode = 'form';
      // Populate default values
      for (const field of this.startFormFields) {
        if (field.type === 'boolean') {
          this.formValues[field.id] = field.defaultValue === 'true';
        } else if (field.type === 'long' || field.type === 'integer') {
          this.formValues[field.id] = field.defaultValue ? Number(field.defaultValue) : 0;
        } else if (field.type === 'enum' && field.values && field.values.length > 0) {
          this.formValues[field.id] = field.defaultValue || field.values[0].id;
        } else {
          this.formValues[field.id] = field.defaultValue || '';
        }
      }
      this.executeInputJson = JSON.stringify(this.formValues, null, 2);
    } else {
      this.hasStartForm = false;
      this.executionInputMode = 'json';
      this.executeInputJson = '{\n  "approved": true\n}';
    }

    this.showExecuteModal = true;
  }

  async openExecuteModalFromForm(): Promise<void> {
    if (this.selectedWorkflow) {
      await this.openExecuteModal(this.selectedWorkflow);
    } else if (this.isEditing && this.editingWorkflowId) {
      const activeXml = this.bpmnDesigner ? await this.bpmnDesigner.getXml() : this.workflowForm.get('payload')?.value;
      const currentWf: WorkflowDefinition = {
        id: this.editingWorkflowId,
        code: this.workflowForm.get('code')?.value,
        name: this.workflowForm.get('name')?.value,
        vnamespace: this.workflowForm.get('vnamespace')?.value,
        version: this.workflowForm.get('version')?.value || 1,
        payloadFormat: 'json',
        payload: activeXml,
        maxDurationSeconds: 3600,
        isActive: true,
        scope: this.scope
      };
      await this.openExecuteModal(currentWf);
    }
  }

  validateField(field: GeneratedFormField): string {
    const val = this.formValues[field.id];
    const valStr = val !== undefined && val !== null ? String(val).trim() : '';
    const normType = (field.type || 'string').toLowerCase();

    // Required check
    if (field.required) {
      if (normType === 'boolean') {
        if (val !== true && val !== 'true' && val !== 1) {
          return `'${field.label || field.id}' field is required.`;
        }
      } else if (!valStr) {
        return `'${field.label || field.id}' field is required.`;
      }
    }

    if (!valStr) return ''; // If not required and empty, skip range/length checks

    // String length & pattern checks
    if (normType === 'string' || normType === 'text' || normType === 'longtext' || normType === '') {
      const minL = field.minlength !== undefined ? Number(field.minlength) : (field.min !== undefined ? Number(field.min) : undefined);
      const maxL = field.maxlength !== undefined ? Number(field.maxlength) : (field.max !== undefined ? Number(field.max) : undefined);

      if (minL !== undefined && !isNaN(minL) && valStr.length < minL) {
        return `'${field.label || field.id}' field must be at least ${minL} characters (current length: ${valStr.length}).`;
      }
      if (maxL !== undefined && !isNaN(maxL) && valStr.length > maxL) {
        return `'${field.label || field.id}' field must be at most ${maxL} characters (current length: ${valStr.length}).`;
      }
      if (field.pattern) {
        try {
          const reg = new RegExp(field.pattern);
          if (!reg.test(valStr)) {
            return `'${field.label || field.id}' field value does not match required pattern (${field.pattern}).`;
          }
        } catch {
          // ignore invalid pattern regex
        }
      }
    }

    // Number min/max checks
    if (normType === 'long' || normType === 'integer' || normType === 'number' || normType === 'float' || normType === 'double') {
      const numVal = Number(val);
      if (isNaN(numVal)) {
        return `'${field.label || field.id}' field must be a valid number.`;
      }
      const minN = field.min !== undefined ? Number(field.min) : (field.minlength !== undefined ? Number(field.minlength) : undefined);
      const maxN = field.max !== undefined ? Number(field.max) : (field.maxlength !== undefined ? Number(field.maxlength) : undefined);

      if (minN !== undefined && !isNaN(minN) && numVal < minN) {
        return `'${field.label || field.id}' field must be greater than or equal to ${minN}.`;
      }
      if (maxN !== undefined && !isNaN(maxN) && numVal > maxN) {
        return `'${field.label || field.id}' field must be less than or equal to ${maxN}.`;
      }
    }

    return '';
  }

  onFormFieldChange(field: GeneratedFormField): void {
    const err = this.validateField(field);
    const newErrors = { ...this.formErrors };
    if (err) {
      newErrors[field.id] = err;
    } else {
      delete newErrors[field.id];
    }
    this.formErrors = newErrors;
  }

  validateAllFormFields(): boolean {
    const newErrors: { [key: string]: string } = {};
    let isValid = true;
    for (const field of this.startFormFields) {
      const err = this.validateField(field);
      if (err) {
        newErrors[field.id] = err;
        isValid = false;
      }
    }
    this.formErrors = newErrors;
    return isValid;
  }

  confirmExecuteWorkflow(): void {
    if (!this.executingWorkflow || !this.executingWorkflow.id) return;

    let inputObj: any = {};
    if (this.hasStartForm && this.executionInputMode === 'form') {
      if (!this.validateAllFormFields()) {
        this.executeErrorMessage = 'Please fix the form errors before continuing.';
        return;
      }

      inputObj = {};
      for (const field of this.startFormFields) {
        const val = this.formValues[field.id];
        if (field.type === 'boolean') {
          inputObj[field.id] = Boolean(val);
        } else if (field.type === 'long' || field.type === 'integer') {
          inputObj[field.id] = val !== '' && val !== null && !isNaN(Number(val)) ? Number(val) : 0;
        } else {
          inputObj[field.id] = val !== undefined && val !== null ? val : '';
        }
      }
      this.executeInputJson = JSON.stringify(inputObj, null, 2);
    } else if (this.executeInputJson && this.executeInputJson.trim()) {
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
    const execPolicy = (this.executingWorkflow?.onVersionChange && this.executingWorkflow.onVersionChange !== 'defined_in_execution')
      ? this.executingWorkflow.onVersionChange
      : (this.executionOnVersionChange || 'continue');

    const exec$ = this.scope === 'global'
      ? this.workflowsService.executeGlobalWorkflow(this.executingWorkflow.id, inputObj, this.executionKey, vns, execPolicy)
      : this.workflowsService.executeTenantWorkflow(this.tenantCode, this.executingWorkflow.id, inputObj, this.executionKey, vns, execPolicy);

    exec$.subscribe({
      next: (res: any) => {
        this.executing = false;
        this.executionResult = res?.Result || res?.result || res;
        this.lastExecutedWorkflow = this.executingWorkflow;
        this.showExecuteModal = false;
        this.showExecutionSuccessModal = true;
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
    this.startFormFields = [];
    this.formValues = {};
    this.formErrors = {};
    this.hasStartForm = false;
  }

  closeSuccessModal(): void {
    this.showExecutionSuccessModal = false;
    this.executionResult = null;
    this.lastExecutedWorkflow = null;
  }

  async runAnotherWorkflow(): Promise<void> {
    const targetWf = this.lastExecutedWorkflow;
    this.showExecutionSuccessModal = false;
    this.executionResult = null;
    if (targetWf) {
      await this.openExecuteModal(targetWf);
    }
  }

  // --- EXECUTIONS LIST HANDLERS ---
  openExecutionsModal(wf: WorkflowDefinition): void {
    this.navigateToExecutions(wf);
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
            payloadSnapshot: rawExec.payloadSnapshot || rawExec.PayloadSnapshot || '',
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

  selectDetailTab(tab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'diagram'): void {
    this.activeDetailTab = tab;
    if (tab === 'diagram') {
      setTimeout(() => this.highlightExecutionDiagram(), 250);
    }
  }

  getExecutionSnapshotPayload(): string {
    if (!this.selectedExecutionDetail?.execution) return DEFAULT_BPMN_XML;
    const snap = this.selectedExecutionDetail.execution.payloadSnapshot;
    if (snap) {
      return this.decodePayload(snap) || DEFAULT_BPMN_XML;
    }
    if (this.selectedWorkflow?.payload) {
      return this.decodePayload(this.selectedWorkflow.payload) || DEFAULT_BPMN_XML;
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
    }

    // Failed jobs -> red highlight
    const jobs = this.selectedExecutionDetail.jobs || [];
    jobs.forEach(j => {
      if (j.activityId && (j.status === 'failed' || j.error)) {
        markers.push({ id: j.activityId, type: 'error' });
      }
    });

    this.executionBpmnDesigner.highlightElements(markers);
  }

  // --- VERSION HISTORY HANDLERS ---
  openVersionHistoryModal(wf: WorkflowDefinition): void {
    this.navigateToVersionHistory(wf);
  }

  loadVersionHistory(): void {
    if (!this.selectedVersionWorkflow?.id) return;
    this.loadingVersionHistory = true;
    this.versionHistoryError = '';

    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflowVersions(this.selectedVersionWorkflow.id)
      : this.workflowsService.getTenantWorkflowVersions(this.tenantCode, this.selectedVersionWorkflow.id);

    req$.subscribe({
      next: (res: any) => {
        const raw = res?.Entities || res?.entities || res?.items || (Array.isArray(res) ? res : []);
        this.versionHistoryList = raw.map((v: any) => ({
          id: v.id || v.ID || '',
          workflowDefinitionId: v.workflowDefinitionId || v.WorkflowDefinitionID || '',
          version: v.version || v.Version || 1,
          vnamespace: v.vnamespace || v.VNamespace || '',
          payload: v.payload || v.Payload || '',
          payloadFormat: v.payloadFormat || v.PayloadFormat || 'bpmn',
          structuralHash: v.structuralHash || v.StructuralHash || '',
          createdAt: v.createdAt || v.CreatedAt || ''
        }));

        if (this.versionHistoryList.length === 0 && this.selectedVersionWorkflow) {
          this.versionHistoryList = [{
            id: this.selectedVersionWorkflow.id || 'v1',
            workflowDefinitionId: this.selectedVersionWorkflow.id || '',
            version: this.selectedVersionWorkflow.version || 1,
            vnamespace: this.selectedVersionWorkflow.vnamespace || '',
            payload: this.selectedVersionWorkflow.payload || '',
            payloadFormat: this.selectedVersionWorkflow.payloadFormat || 'bpmn',
            structuralHash: 'Current (Initial Version)',
            createdAt: this.selectedVersionWorkflow.updatedAt || this.selectedVersionWorkflow.createdAt || new Date().toISOString()
          }];
        } else if (this.selectedVersionWorkflow && this.versionHistoryList.length > 0) {
          const minVersion = Math.min(...this.versionHistoryList.map(v => v.version));
          if (minVersion > 1) {
            this.versionHistoryList.push({
              id: `${this.selectedVersionWorkflow.id}-v1`,
              workflowDefinitionId: this.selectedVersionWorkflow.id || '',
              version: 1,
              vnamespace: this.selectedVersionWorkflow.vnamespace || '',
              payload: this.selectedVersionWorkflow.payload || '',
              payloadFormat: this.selectedVersionWorkflow.payloadFormat || 'bpmn',
              structuralHash: 'Current (Initial Version)',
              createdAt: this.selectedVersionWorkflow.createdAt || new Date().toISOString()
            });
          }
        }

        this.versionHistoryList.sort((a, b) => b.version - a.version);
        this.loadingVersionHistory = false;
      },
      error: (err) => {
        this.versionHistoryError = ErrorUtil.formatErrorMessage(err);
        this.loadingVersionHistory = false;
      }
    });
  }

  closeVersionHistoryModal(): void {
    this.showVersionHistoryModal = false;
    this.selectedVersionWorkflow = null;
    this.versionHistoryList = [];
  }

  @ViewChild('versionBpmnDesigner') versionBpmnDesigner?: BpmnDesignerComponent;

  openVersionDiagramModal(ver: WorkflowDefinitionVersion): void {
    this.selectedVersionRecord = ver;

    const loadPayloadAndOpen = (payloadRaw: any) => {
      this.versionDiagramPayload = this.decodePayload(payloadRaw) || DEFAULT_BPMN_XML;
      this.showVersionDiagramModal = true;
      setTimeout(() => {
        if (this.versionBpmnDesigner) {
          this.versionBpmnDesigner.refresh();
        }
      }, 150);
    };

    if (!ver.payload || (typeof ver.payload === 'string' && !ver.payload.trim())) {
      const req$ = this.scope === 'global'
        ? this.workflowsService.getGlobalWorkflowVersion(ver.workflowDefinitionId, ver.version)
        : this.workflowsService.getTenantWorkflowVersion(this.tenantCode, ver.workflowDefinitionId, ver.version);

      req$.subscribe({
        next: (res: any) => {
          const fetchedPayload = res?.payload || res?.Payload || ver.payload;
          loadPayloadAndOpen(fetchedPayload);
        },
        error: () => {
          loadPayloadAndOpen(ver.payload);
        }
      });
    } else {
      loadPayloadAndOpen(ver.payload);
    }
  }

  closeVersionDiagramModal(): void {
    this.showVersionDiagramModal = false;
    this.selectedVersionRecord = null;
    this.versionDiagramPayload = '';
  }
}
