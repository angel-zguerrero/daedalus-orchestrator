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
import { UsersService } from '../users/services/users.service';

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
  selectedWorkflowUserName: string = '';
  payloadDisplayText: string = '';

  // Delete Confirm Modal
  showDeleteModal: boolean = false;
  workflowToDelete: WorkflowDefinition | null = null;

  // Execute Workflow Modal
  showExecuteModal: boolean = false;
  showExecutionSuccessModal: boolean = false;
  executingWorkflow: WorkflowDefinition | null = null;
  lastExecutedWorkflow: WorkflowDefinition | null = null;
  executeInputJson: string = '{}';
  executionKey: string = '';
  executionOnVersionChange: string = 'continue';
  executing: boolean = false;
  executionResult: any = null;
  executeErrorMessage: string = '';
  startFormFields: GeneratedFormField[] = [];
  startExtensionProperties: Record<string, string> = {};
  hasStartExtensionProperties: boolean = false;
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
  selectedExecutionUserName: string = '';
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
    private usersService: UsersService,
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
      userId: w.userId || w.UserID || '',
      userName: w.userName || w.UserName || '',
      accountId: w.accountId || w.AccountID || '',
      accountName: w.accountName || w.AccountName || '',
      externalUserId: w.externalUserId || w.ExternalUserID || '',
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

        const userIds = this.workflows.map(w => w.userId).filter((id): id is string => !!id);
        if (userIds.length > 0) {
          this.usersService.resolveUsers(userIds).subscribe(userMap => {
            this.workflows.forEach(w => {
              if (w.userId && userMap.has(w.userId)) {
                w.userName = userMap.get(w.userId);
              }
            });
          });
        }

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
    this.selectedWorkflowUserName = wf.userName || wf.userId || '';
    if (wf.userId) {
      this.usersService.resolveUser(wf.userId).subscribe(name => {
        this.selectedWorkflowUserName = name;
      });
    }
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

      // Validate diagram form fields - do not allow saving if form fields have missing IDs
      const formValidation = BpmnFormParserUtil.validateDiagramFormFields(xml || '');
      if (!formValidation.isValid) {
        this.errorMessage = `Cannot save workflow diagram: ${formValidation.errors.join(' ')}`;
        this.showAlert = true;
        return;
      }
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

    // Extract Generated Task Form fields and Extension Properties from StartEvent
    this.startFormFields = BpmnFormParserUtil.extractStartFormFields(xmlPayload);
    this.startExtensionProperties = BpmnFormParserUtil.extractStartExtensionProperties(xmlPayload);
    this.hasStartExtensionProperties = Object.keys(this.startExtensionProperties).length > 0;
    this.hasStartForm = this.startFormFields.length > 0;
    this.formValues = {};
    this.formErrors = {};

    // Business rule:
    // - Extension properties present → JSON view (shows the required JSON schema)
    // - Only form fields present → Form view with validations
    // - Neither → JSON view (empty object)
    if (this.hasStartExtensionProperties) {
      this.executionInputMode = 'json';
    } else if (this.hasStartForm) {
      this.executionInputMode = 'form';
    } else {
      this.executionInputMode = 'json';
    }

    // Populate default form values
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

    if (this.hasStartForm || this.hasStartExtensionProperties) {
      this.executeInputJson = this.generateCombinedJsonPayload();
    } else {
      this.executeInputJson = '{}';
    }

    this.showExecuteModal = true;
  }

  hasExtensionProperties(): boolean {
    return this.hasStartExtensionProperties;
  }

  hasFormFields(): boolean {
    return this.hasStartForm;
  }

  getExtensionProperties(): Record<string, string> {
    return this.startExtensionProperties;
  }

  getExtensionPropertyList(): Array<{ key: string; value: string }> {
    return Object.entries(this.startExtensionProperties).map(([key, value]) => ({ key, value }));
  }

  /**
   * Sets a value at a dot-notation path in the given object (e.g. "a.b.c" → obj.a.b.c = value).
   */
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

  /**
   * Resolves a dot-notation path from an object (e.g. "a.b.c" → obj.a.b.c).
   */
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

  /**
   * Generates the combined JSON payload from form fields + extension properties.
   * Form fields contribute their current values; extension property keys are
   * scaffolded with empty strings (user fills them in the JSON editor).
   */
  generateCombinedJsonPayload(): string {
    const combined: Record<string, any> = {};

    // 1. Add form fields with their default/current values
    for (const f of this.startFormFields) {
      let defaultVal: any = '';
      if (f.defaultValue !== undefined && f.defaultValue !== null && f.defaultValue !== '') {
        if (f.type === 'boolean' || f.type === 'bool') {
          defaultVal = f.defaultValue === 'true';
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
      if (this.formValues && this.formValues[f.id] !== undefined) {
        defaultVal = this.formValues[f.id];
      }
      this.setNestedProperty(combined, f.id, defaultVal);
    }

    // 2. Scaffold extension property keys (dot-notation supported, values left empty)
    for (const k of Object.keys(this.startExtensionProperties)) {
      this.setNestedProperty(combined, k, '');
    }

    return JSON.stringify(combined, null, 2);
  }

  /**
   * Switches between 'form' and 'json' execution input modes,
   * syncing values between the two representations.
   */
  setExecutionInputMode(mode: 'form' | 'json'): void {
    if (mode === 'json' && this.executionInputMode === 'form') {
      this.executeInputJson = this.generateCombinedJsonPayload();
    } else if (mode === 'form' && this.executionInputMode === 'json') {
      try {
        const parsed = JSON.parse(this.executeInputJson || '{}');
        for (const f of this.startFormFields) {
          const val = this.resolveVariablePathInObject(f.id, parsed);
          if (val !== undefined) {
            this.formValues[f.id] = val;
          }
        }
      } catch {
        // keep existing form values if JSON parse fails
      }
    }
    this.executionInputMode = mode;
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

      try {
        inputObj = JSON.parse(this.generateCombinedJsonPayload());
      } catch {
        inputObj = {};
      }

      for (const field of this.startFormFields) {
        const val = this.formValues[field.id];
        let fieldVal: any;
        if (field.type === 'boolean') {
          fieldVal = Boolean(val);
        } else if (field.type === 'long' || field.type === 'integer') {
          fieldVal = val !== '' && val !== null && !isNaN(Number(val)) ? Number(val) : 0;
        } else {
          fieldVal = val !== undefined && val !== null ? val : '';
        }
        this.setNestedProperty(inputObj, field.id, fieldVal);
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
        const raw = res?.Result || res?.result || res;
        this.executionResult = raw ? {
          ...raw,
          id: raw.id || raw.ID || raw.executionKey || raw.ExecutionKey || '',
          executionKey: raw.executionKey || raw.ExecutionKey || raw.id || raw.ID || '',
          workflowDefinitionId: raw.workflowDefinitionId || raw.WorkflowDefinitionID || this.executingWorkflow?.id || '',
          status: raw.status || raw.Status || 'running',
          startedAt: raw.startedAt || raw.StartedAt || raw.createdAt || raw.CreatedAt || ''
        } : null;
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
    this.startExtensionProperties = {};
    this.hasStartExtensionProperties = false;
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
    this.selectedExecutionUserName = '';
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
          }))
        };

        this.selectedExecutionUserName = this.selectedExecutionDetail.execution.userName || this.selectedExecutionDetail.execution.userId || '';
        if (this.selectedExecutionDetail.execution.userId) {
          this.usersService.resolveUser(this.selectedExecutionDetail.execution.userId).subscribe(name => {
            this.selectedExecutionUserName = name;
          });
        }

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
