import { Component, OnInit, ViewChild, HostListener } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { map, switchMap, startWith, debounceTime } from 'rxjs/operators';
import {
  ButtonModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ModalModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition } from '../services/workflows.service';
import { VNamespacesService } from '../../tenants/tenant-management/services/vnamespaces.service';
import { ErrorUtil } from '../../../shared/utils/error.util';
import { BpmnDesignerComponent, DEFAULT_BPMN_XML } from '../../../shared/components/bpmn-designer/bpmn-designer.component';
import { DesignErrorsConsoleComponent } from '../../../shared/components/design-errors-console/design-errors-console.component';
import { ComponentWithUnsavedChanges } from '../guards/workflow-unsaved.guard';

@Component({
  selector: 'app-workflow-editor',
  templateUrl: './workflow-editor.component.html',
  styleUrls: ['./workflow-editor.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    RouterModule,
    ButtonModule,
    CardModule,
    FormModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    ModalModule,
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    BpmnDesignerComponent
  ]
})
export class WorkflowEditorComponent implements OnInit, ComponentWithUnsavedChanges {
  isEditing: boolean = false;
  workflowId: string = '';
  scope: 'global' | 'tenant' = 'global';
  tenantCode: string = '';
  currentWorkflowVersion: number = 1;

  loading: boolean = false;
  saving: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  showAdvancedSettings: boolean = false;
  showUnsavedConfirmModal: boolean = false;
  hasUnsavedChanges: boolean = false;
  pendingNavigation: boolean = false;

  workflowForm: FormGroup;
  vnamespaceCtrl = new FormControl('default');
  filteredVNamespaces!: Observable<any[]>;
  loadingVNamespaces: boolean = false;

  @ViewChild('bpmnDesigner') bpmnDesigner?: BpmnDesignerComponent;

  constructor(
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router,
    private location: Location,
    private workflowsService: WorkflowsService,
    private vNamespacesService: VNamespacesService
  ) {
    this.workflowForm = this.fb.group({
      name: ['Untitled Workflow', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      vnamespace: this.vnamespaceCtrl,
      description: [''],
      onVersionChange: ['defined_in_execution', Validators.required],
      payloadFormat: ['bpmn', Validators.required],
      payload: [DEFAULT_BPMN_XML],
      maxDurationSeconds: [3600, [Validators.required, Validators.min(0)]],
      isActive: [true]
    });

    this.workflowForm.valueChanges.subscribe(() => {
      if (!this.loading) {
        this.hasUnsavedChanges = true;
      }
    });

    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );
  }

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
      if (params['id']) {
        this.isEditing = true;
        this.workflowId = params['id'];
        this.loadWorkflow(this.workflowId);
      } else {
        this.isEditing = false;
        this.hasUnsavedChanges = false;
        setTimeout(() => {
          if (this.bpmnDesigner) {
            this.bpmnDesigner.refresh();
          }
        }, 150);
      }
    });
  }

  @HostListener('window:beforeunload', ['$event'])
  onBeforeUnload(event: BeforeUnloadEvent): boolean | void {
    if (this.hasUnsavedChanges) {
      event.preventDefault();
      event.returnValue = '';
      return false;
    }
  }

  canDeactivate(): boolean | Observable<boolean> | Promise<boolean> {
    if (!this.hasUnsavedChanges) {
      return true;
    }
    return new Promise<boolean>((resolve) => {
      this.showUnsavedConfirmModal = true;
      this.deactivateSubject = resolve;
    });
  }

  private deactivateSubject?: (canDeactivate: boolean) => void;

  toggleAdvancedSettings(): void {
    this.showAdvancedSettings = !this.showAdvancedSettings;
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

  private _filterVNamespaces(value: string): Observable<any[]> {
    if (!this.tenantCode) return of([]);
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantCode, '', 20, value).pipe(
      map(response => {
        this.loadingVNamespaces = false;
        return response.data || response.result?.Entities || response.entities || [];
      })
    );
  }

  loadWorkflow(id: string): void {
    this.loading = true;
    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflow(id)
      : this.workflowsService.getTenantWorkflow(this.tenantCode, id);

    req$.subscribe({
      next: (wf: any) => {
        const entity = wf.Entity || wf.entity || wf;
        this.currentWorkflowVersion = entity.version || entity.Version || 1;
        const decodedPayload = this.decodePayload(entity.payload || entity.Payload);

        this.workflowForm.patchValue({
          code: entity.code || entity.Code || '',
          name: entity.name || entity.Name || 'Untitled Workflow',
          vnamespace: this.scope === 'global' ? '' : (entity.vnamespace || entity.VNamespace || 'default'),
          description: entity.description || entity.Description || '',
          onVersionChange: entity.onVersionChange || entity.OnVersionChange || 'defined_in_execution',
          payloadFormat: 'bpmn',
          payload: decodedPayload || DEFAULT_BPMN_XML,
          maxDurationSeconds: entity.maxDurationSeconds ?? entity.MaxDurationSeconds ?? 3600,
          isActive: entity.isActive !== undefined ? entity.isActive : (entity.IsActive !== undefined ? entity.IsActive : true)
        });
        this.workflowForm.get('code')?.disable();
        this.hasUnsavedChanges = false;
        this.loading = false;

        setTimeout(() => {
          if (this.bpmnDesigner) {
            this.bpmnDesigner.refresh();
          }
        }, 150);
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.loading = false;
      }
    });
  }

  onBpmnXmlChange(xml: string): void {
    this.workflowForm.patchValue({ payload: xml }, { emitEvent: false });
    if (!this.loading) {
      this.hasUnsavedChanges = true;
    }
  }

  cancelClose(): void {
    if (this.hasUnsavedChanges) {
      this.showUnsavedConfirmModal = true;
    } else {
      this.navigateBack();
    }
  }

  cancelCloseUnsavedModal(): void {
    this.showUnsavedConfirmModal = false;
    if (this.deactivateSubject) {
      this.deactivateSubject(false);
      this.deactivateSubject = undefined;
    }
  }

  confirmDiscardAndClose(): void {
    this.hasUnsavedChanges = false;
    this.showUnsavedConfirmModal = false;
    if (this.deactivateSubject) {
      this.deactivateSubject(true);
      this.deactivateSubject = undefined;
    } else {
      this.navigateBack();
    }
  }

  async saveWorkflowAndClose(): Promise<void> {
    await this.saveWorkflow(true);
  }

  async saveWorkflow(closeAfterSave: boolean = false): Promise<void> {
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

    this.saving = true;
    this.showAlert = false;

    if (this.isEditing) {
      const update$ = this.scope === 'global'
        ? this.workflowsService.updateGlobalWorkflow(this.workflowId, payload)
        : this.workflowsService.updateTenantWorkflow(this.tenantCode, this.workflowId, payload);

      update$.subscribe({
        next: (res: any) => {
          this.saving = false;
          this.hasUnsavedChanges = false;
          const updatedEntity = res?.Entity || res?.entity || res;
          if (updatedEntity && updatedEntity.version) {
            this.currentWorkflowVersion = updatedEntity.version;
          }
          this.successMsg = 'Workflow definition updated successfully.';
          setTimeout(() => this.successMsg = '', 3000);

          if (closeAfterSave) {
            if (this.deactivateSubject) {
              this.deactivateSubject(true);
              this.deactivateSubject = undefined;
            }
            this.navigateBack();
          }
        },
        error: (err) => {
          this.saving = false;
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
          this.saving = false;
          this.hasUnsavedChanges = false;
          const createdEntity = res?.Entity || res?.entity || res;
          if (createdEntity && createdEntity.id) {
            this.isEditing = true;
            this.workflowId = createdEntity.id;
            if (createdEntity.version) {
              this.currentWorkflowVersion = createdEntity.version;
            }
            this.workflowForm.get('code')?.disable();
          }
          this.successMsg = 'Workflow definition created successfully.';
          setTimeout(() => this.successMsg = '', 3000);

          if (closeAfterSave) {
            if (this.deactivateSubject) {
              this.deactivateSubject(true);
              this.deactivateSubject = undefined;
            }
            this.navigateBack();
          }
        },
        error: (err) => {
          this.saving = false;
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    }
  }

  navigateBack(): void {
    if (this.tenantCode) {
      this.router.navigate(['/tenants', this.tenantCode, 'management'], { queryParams: { tab: 'workflows' } });
    } else {
      this.router.navigate(['/workflows']);
    }
  }
}
