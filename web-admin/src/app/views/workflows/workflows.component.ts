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
import { WorkflowsService, WorkflowDefinition } from './services/workflows.service';
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
  workflowForm: FormGroup;
  isEditing: boolean = false;
  editingWorkflowId: string = '';
  @ViewChild('bpmnDesigner') bpmnDesigner?: BpmnDesignerComponent;

  // Detail / Payload Modal
  showDetailModal: boolean = false;
  selectedWorkflow: WorkflowDefinition | null = null;
  payloadDisplayText: string = '';

  // Delete Confirm Modal
  showDeleteModal: boolean = false;
  workflowToDelete: WorkflowDefinition | null = null;

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
    this.workflowForm.reset({
      vnamespace: 'default',
      version: 1,
      payloadFormat: 'bpmn',
      payload: DEFAULT_BPMN_XML,
      maxDurationSeconds: 3600,
      isActive: true
    });
    this.workflowForm.get('code')?.enable();
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
    const decodedPayload = this.decodePayload(wf.payload);

    this.workflowForm.patchValue({
      code: wf.code,
      name: wf.name,
      vnamespace: wf.vnamespace || 'default',
      description: wf.description,
      version: wf.version,
      payloadFormat: 'bpmn',
      payload: decodedPayload || DEFAULT_BPMN_XML,
      maxDurationSeconds: wf.maxDurationSeconds,
      isActive: wf.isActive
    });
    this.workflowForm.get('code')?.disable();
    this.showModal = true;
    setTimeout(() => {
      if (this.bpmnDesigner) {
        this.bpmnDesigner.refresh();
      }
    }, 150);
  }

  onBpmnXmlChange(xml: string): void {
    this.workflowForm.patchValue({ payload: xml }, { emitEvent: false });
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

  async saveWorkflow(): Promise<void> {
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
      vnamespace: val.vnamespace ? val.vnamespace.trim() : 'default',
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
          this.showModal = false;
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
        next: () => {
          this.showModal = false;
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
}
