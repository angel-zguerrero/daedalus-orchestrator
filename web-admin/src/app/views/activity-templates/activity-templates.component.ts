import { Component, OnInit, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { map, switchMap, startWith, debounceTime, catchError } from 'rxjs/operators';
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
  BadgeComponent
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { ActivityTemplatesService, ActivityTemplate } from './services/activity-templates.service';
import { VNamespacesService } from '../tenants/tenant-management/services/vnamespaces.service';
import { ErrorUtil } from '../../shared/utils/error.util';
import elementTemplatesData from '../../shared/components/bpmn-designer/element-templates.json';

export interface PreviewProperty {
  index: number;
  label: string;
  type: string;
  multiline?: boolean;
  value: any;
  editable: boolean;
  bindingName: string;
  group?: string;
  choices?: { name: string; value: string }[];
  notEmpty?: boolean;
}

export interface PreviewGroup {
  id: string;
  label: string;
  properties: PreviewProperty[];
}

export interface ParentTemplateOption {
  id: string;
  name: string;
  description: string;
  isBuiltin: boolean;
  rootActivity: string;
  scope?: string;
  rawTemplate: any;
}

@Component({
  selector: 'app-activity-templates',
  templateUrl: './activity-templates.component.html',
  styleUrls: ['./activity-templates.component.scss'],
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
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule
  ]
})
export class ActivityTemplatesComponent implements OnInit, OnChanges {
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';

  templates: ActivityTemplate[] = [];
  filteredTemplates: ActivityTemplate[] = [];
  loading: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  searchQuery: string = '';

  // Pagination
  cursor: string = '';
  nextCursor: string = '';
  cursors: string[] = [''];
  pageSize: number = 20;

  // Create / Edit 2-Step Modal
  showModal: boolean = false;
  currentStep: 'identity' | 'designer' = 'identity';
  showAdvancedSettings: boolean = false;
  isEditing: boolean = false;
  editingTemplateId: string = '';

  identityForm: FormGroup;
  designerForm: FormGroup;

  // Activity Family Autocomplete (handled identically to VNamespace)
  activityFamilyCtrl = new FormControl('default', Validators.required);
  filteredActivityFamilies!: Observable<string[]>;
  activityFamilyFilterCtrl = new FormControl('');
  filteredFilterActivityFamilies!: Observable<string[]>;
  selectedActivityFamilyFilter: string = '';
  knownFamilies: string[] = ['default', 'cache', 'http', 'logging', 'messaging', 'database'];

  // VNamespace Autocomplete (for tenant scope, same as Workflows)
  vnamespaceCtrl = new FormControl('default');
  filteredVNamespaces!: Observable<any[]>;
  vnamespaceFilterCtrl = new FormControl('');
  filteredFilterVNamespaces!: Observable<any[]>;
  selectedVNamespaceFilter: string = '';

  // Parent Templates (Built-in + existing custom templates)
  builtinOptions: ParentTemplateOption[] = [];
  customParentOptions: ParentTemplateOption[] = [];
  selectedParentOption: ParentTemplateOption | null = null;

  // Designer Payload & Live Preview
  payloadJson: string = '';
  previewGroups: PreviewGroup[] = [];
  validationErrors: string[] = [];
  validationPassed: boolean = false;

  // Details & Delete Modals
  showDetailModal: boolean = false;
  selectedTemplate: ActivityTemplate | null = null;
  detailPayloadText: string = '';

  showDeleteModal: boolean = false;
  templateToDelete: ActivityTemplate | null = null;

  constructor(
    private fb: FormBuilder,
    private activityTemplatesService: ActivityTemplatesService,
    private vNamespacesService: VNamespacesService
  ) {
    this.initBuiltinTemplates();

    this.identityForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      activityFamily: this.activityFamilyCtrl,
      vnamespace: this.vnamespaceCtrl,
      parentTemplateId: ['', Validators.required]
    });

    this.designerForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[a-z0-9-]*$')]],
      description: [''],
      activityFamily: this.activityFamilyCtrl,
      vnamespace: this.vnamespaceCtrl,
      parentTemplateId: ['', Validators.required],
      rootActivity: [''],
      isActive: [true]
    });

    // Activity Family Autocomplete streams
    this.filteredActivityFamilies = this.activityFamilyCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(200),
      map((val) => this._filterFamilies(val || ''))
    );

    this.filteredFilterActivityFamilies = this.activityFamilyFilterCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(200),
      map((val) => this._filterFamilies(val || ''))
    );

    // VNamespace Autocomplete streams
    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap((value) => this._filterVNamespaces(value || ''))
    );

    this.filteredFilterVNamespaces = this.vnamespaceFilterCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap((value) => this._filterVNamespaces(value || ''))
    );
  }

  ngOnInit(): void {
    this.loadTemplates();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['tenantCode'] || changes['scope']) {
      this.cursor = '';
      this.cursors = [''];
      this.loadTemplates();
    }
  }

  private initBuiltinTemplates(): void {
    this.builtinOptions = (elementTemplatesData as any[]).map((tpl) => ({
      id: tpl.id,
      name: tpl.name,
      description: tpl.description || '',
      isBuiltin: true,
      rootActivity: tpl.id,
      rawTemplate: JSON.parse(JSON.stringify(tpl))
    }));
  }

  private _filterFamilies(query: string): string[] {
    const q = (query || '').toLowerCase().trim();
    const all = Array.from(new Set([...this.knownFamilies]));
    if (!q) return all;
    return all.filter((f) => f.toLowerCase().includes(q));
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    if (this.scope === 'global' || !this.tenantCode) {
      return of([{ Name: 'default' }]);
    }
    return this.vNamespacesService.getVNamespaces(this.tenantCode, '', 20, value).pipe(
      map((response: any) => {
        const list = response?.Entities || response?.entities || [];
        return list.length > 0 ? list : [{ Name: 'default' }];
      }),
      catchError(() => of([{ Name: 'default' }]))
    );
  }

  private toKebabCase(str: string): string {
    return (str || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
  }

  loadTemplates(): void {
    this.loading = true;
    this.showAlert = false;

    const obs =
      this.scope === 'global'
        ? this.activityTemplatesService.getGlobalTemplates(
            this.pageSize,
            this.cursor,
            '',
            this.selectedActivityFamilyFilter
          )
        : this.activityTemplatesService.getTenantTemplates(
            this.tenantCode,
            this.pageSize,
            this.cursor,
            this.selectedVNamespaceFilter,
            this.selectedActivityFamilyFilter
          );

    obs.subscribe({
      next: (res: any) => {
        const list: ActivityTemplate[] = res?.Entities || res?.entities || [];
        this.templates = list;
        this.nextCursor = res?.Cursor || res?.cursor || '';
        this.updateKnownFamiliesAndCustomParents(list);
        this.applyClientFilters();
        this.loading = false;
      },
      error: (err: any) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.loading = false;
      }
    });
  }

  private updateKnownFamiliesAndCustomParents(list: ActivityTemplate[]): void {
    const famSet = new Set<string>(this.knownFamilies);
    list.forEach((t) => {
      if (t.activityFamily) {
        famSet.add(t.activityFamily);
      }
    });
    this.knownFamilies = Array.from(famSet);

    // Also load available templates for inheritance (Global + current Tenant if tenant scope)
    this.activityTemplatesService
      .getForDesigner(this.scope === 'tenant' ? this.tenantCode : '', 50, '')
      .subscribe({
        next: (res: any) => {
          const entities: ActivityTemplate[] = res?.Entities || res?.entities || [];
          this.customParentOptions = entities
            .map((t) => {
              const decoded = this.activityTemplatesService.decodePayload(t.payload);
              let parsed: any = null;
              try {
                parsed = JSON.parse(decoded);
              } catch {
                parsed = null;
              }
              if (!parsed) return null;
              return {
                id: t.code || t.id || parsed.id,
                name: `${t.name} (${t.scope === 'global' ? 'Global' : 'Tenant'})`,
                description: t.description || parsed.description || '',
                isBuiltin: false,
                rootActivity: t.rootActivity || t.parentTemplateId || parsed.id,
                scope: t.scope,
                rawTemplate: parsed
              } as ParentTemplateOption;
            })
            .filter((x): x is ParentTemplateOption => x !== null);
        }
      });
  }

  applyClientFilters(): void {
    const q = (this.searchQuery || '').toLowerCase().trim();
    this.filteredTemplates = this.templates.filter((t) => {
      const matchesSearch =
        !q ||
        (t.name && t.name.toLowerCase().includes(q)) ||
        (t.code && t.code.toLowerCase().includes(q)) ||
        (t.activityFamily && t.activityFamily.toLowerCase().includes(q));
      return matchesSearch;
    });
  }

  searchTemplates(): void {
    this.applyClientFilters();
  }

  onActivityFamilyFilterChange(value: string): void {
    this.selectedActivityFamilyFilter = (value || '').trim();
    this.cursor = '';
    this.cursors = [''];
    this.loadTemplates();
  }

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = (value || '').trim();
    this.cursor = '';
    this.cursors = [''];
    this.loadTemplates();
  }

  getParentDisplayName(parentId: string): string {
    const foundBuiltin = this.builtinOptions.find(
      (b) => b.id.toLowerCase() === (parentId || '').toLowerCase()
    );
    if (foundBuiltin) return foundBuiltin.name;
    const foundCustom = this.customParentOptions.find(
      (c) => c.id.toLowerCase() === (parentId || '').toLowerCase()
    );
    if (foundCustom) return foundCustom.name;
    return parentId || '-';
  }

  // --- STEP 1: OPEN CREATE MODAL (IDENTITY & BASE TEMPLATE SELECTION) ---
  openCreateModal(): void {
    this.isEditing = false;
    this.editingTemplateId = '';
    this.currentStep = 'identity';
    this.showAdvancedSettings = false;
    this.validationErrors = [];
    this.validationPassed = false;

    this.updateKnownFamiliesAndCustomParents(this.templates);

    this.activityFamilyCtrl.setValue('default');
    this.vnamespaceCtrl.setValue('default');

    const defaultParent = this.builtinOptions[1] || this.builtinOptions[0]; // Default to Redis or HTTP
    this.selectedParentOption = defaultParent || null;

    this.identityForm.reset({
      name: '',
      code: '',
      activityFamily: 'default',
      vnamespace: 'default',
      parentTemplateId: defaultParent ? defaultParent.id : ''
    });

    this.showModal = true;
  }

  onParentSelectionChange(parentId: string): void {
    const allOptions = [...this.builtinOptions, ...this.customParentOptions];
    this.selectedParentOption = allOptions.find((o) => o.id === parentId) || null;
  }

  // Transition from Step 1 (Identity) -> Step 2 (Activity Template Designer)
  proceedToDesigner(): void {
    if (!this.identityForm.get('name')?.value?.trim()) {
      return;
    }
    const parentId = this.identityForm.get('parentTemplateId')?.value;
    this.onParentSelectionChange(parentId);
    if (!this.selectedParentOption) {
      return;
    }

    const name = this.identityForm.get('name')?.value.trim();
    const rawCode = this.identityForm.get('code')?.value?.trim();
    const code = rawCode || this.toKebabCase(name);
    const family = (this.activityFamilyCtrl.value || 'default').trim();
    const vns = (this.vnamespaceCtrl.value || 'default').trim();

    const inheritedRootActivity = this.selectedParentOption.rootActivity || this.selectedParentOption.id;

    this.designerForm.patchValue({
      name,
      code,
      description: `Custom activity template inheriting from ${this.selectedParentOption.name}`,
      activityFamily: family,
      vnamespace: vns,
      parentTemplateId: this.selectedParentOption.id,
      rootActivity: inheritedRootActivity,
      isActive: true
    });

    // Clone the parent's element template JSON and customize id/name/category
    const baseJson = JSON.parse(JSON.stringify(this.selectedParentOption.rawTemplate));
    baseJson.id = code;
    baseJson.name = name;
    baseJson.description = `Custom ${this.selectedParentOption.name} template (${family})`;
    baseJson.category = {
      id: family,
      name: `Family: ${family}`
    };

    // Ensure every property has an explicit editable flag (true by default unless already false in parent)
    if (Array.isArray(baseJson.properties)) {
      baseJson.properties = baseJson.properties.map((p: any) => ({
        ...p,
        editable: p.editable !== undefined ? p.editable : true
      }));
    }

    this.payloadJson = JSON.stringify(baseJson, null, 2);
    this.parsePreviewGroups();
    this.validateTemplate();
    this.currentStep = 'designer';
  }

  // --- OPEN EDIT MODAL (Direct to Step 2 Designer) ---
  openEditModal(tpl: ActivityTemplate): void {
    this.isEditing = true;
    this.editingTemplateId = tpl.id || '';
    this.currentStep = 'designer';
    this.showAdvancedSettings = false;
    this.validationErrors = [];
    this.validationPassed = false;

    this.activityFamilyCtrl.setValue(tpl.activityFamily || 'default');
    this.vnamespaceCtrl.setValue(tpl.vnamespace || 'default');

    this.designerForm.patchValue({
      name: tpl.name,
      code: tpl.code,
      description: tpl.description || '',
      activityFamily: tpl.activityFamily || 'default',
      vnamespace: tpl.vnamespace || 'default',
      parentTemplateId: tpl.parentTemplateId,
      rootActivity: tpl.rootActivity || tpl.parentTemplateId,
      isActive: tpl.isActive
    });

    this.payloadJson = this.activityTemplatesService.decodePayload(tpl.payload);
    this.parsePreviewGroups();
    this.validateTemplate();
    this.showModal = true;
  }

  toggleAdvancedSettings(): void {
    this.showAdvancedSettings = !this.showAdvancedSettings;
  }

  onDesignerNameOrCodeChange(): void {
    try {
      const parsed = JSON.parse(this.payloadJson);
      const name = this.designerForm.get('name')?.value?.trim();
      const code = this.designerForm.get('code')?.value?.trim();
      const family = (this.activityFamilyCtrl.value || 'default').trim();
      if (name) parsed.name = name;
      if (code) parsed.id = code;
      if (family) {
        parsed.category = { id: family, name: `Family: ${family}` };
      }
      this.payloadJson = JSON.stringify(parsed, null, 2);
    } catch {
      // Ignore if JSON is mid-edit
    }
  }

  onPayloadChange(newJson: string): void {
    this.payloadJson = newJson;
    this.parsePreviewGroups();
    this.validateTemplate();
  }

  // Parse JSON into visual property groups for live preview & interactive freeze/burn
  parsePreviewGroups(): void {
    try {
      const parsed = JSON.parse(this.payloadJson);
      const groupsDef: { id: string; label: string }[] = Array.isArray(parsed.groups)
        ? parsed.groups
        : [];
      const propsDef: any[] = Array.isArray(parsed.properties) ? parsed.properties : [];

      const groupMap = new Map<string, PreviewGroup>();
      groupsDef.forEach((g) => {
        groupMap.set(g.id, {
          id: g.id,
          label: g.label || g.id,
          properties: []
        });
      });

      const ungrouped: PreviewGroup = {
        id: '_general',
        label: 'General Properties',
        properties: []
      };

      propsDef.forEach((p, idx) => {
        const previewProp: PreviewProperty = {
          index: idx,
          label: p.label || p.binding?.name || `Property #${idx + 1}`,
          type: p.type || 'String',
          multiline: !!p.multiline,
          value: p.value !== undefined ? p.value : '',
          editable: p.editable !== false && (p.type || '').toLowerCase() !== 'hidden',
          bindingName: p.binding?.name || '',
          group: p.group,
          choices: p.choices || [],
          notEmpty: !!p.constraints?.notEmpty
        };

        if (p.group && groupMap.has(p.group)) {
          groupMap.get(p.group)!.properties.push(previewProp);
        } else {
          ungrouped.properties.push(previewProp);
        }
      });

      const result = Array.from(groupMap.values()).filter((g) => g.properties.length > 0);
      if (ungrouped.properties.length > 0) {
        result.push(ungrouped);
      }
      this.previewGroups = result;
    } catch {
      // Keep previous preview if JSON is temporarily invalid while typing
    }
  }

  // Interactive helper: update property value directly from the visual preview panel
  onPreviewPropertyValueChange(prop: PreviewProperty, newValue: any): void {
    try {
      const parsed = JSON.parse(this.payloadJson);
      if (Array.isArray(parsed.properties) && parsed.properties[prop.index]) {
        parsed.properties[prop.index].value = newValue;
        this.payloadJson = JSON.stringify(parsed, null, 2);
        prop.value = newValue;
        this.validateTemplate();
      }
    } catch {
      // Ignore if JSON invalid
    }
  }

  // Interactive helper: toggle "Burn / Lock" (editable: false) for a property (e.g. Redis connectionString)
  toggleBurnProperty(prop: PreviewProperty): void {
    try {
      const parsed = JSON.parse(this.payloadJson);
      if (Array.isArray(parsed.properties) && parsed.properties[prop.index]) {
        const nextEditable = !prop.editable;
        parsed.properties[prop.index].editable = nextEditable;
        this.payloadJson = JSON.stringify(parsed, null, 2);
        prop.editable = nextEditable;
        this.validateTemplate();
      }
    } catch {
      // Ignore if JSON invalid
    }
  }

  validateTemplate(): boolean {
    const errors: string[] = [];
    try {
      const parsed = JSON.parse(this.payloadJson);
      if (!parsed || typeof parsed !== 'object') {
        errors.push('Template payload must be a valid JSON object.');
      } else {
        if (!parsed.id || typeof parsed.id !== 'string') {
          errors.push('Missing required "id" field in template definition.');
        }
        if (!parsed.name || typeof parsed.name !== 'string') {
          errors.push('Missing required "name" field in template definition.');
        }
        if (!Array.isArray(parsed.appliesTo) || parsed.appliesTo.length === 0) {
          errors.push('Template must specify "appliesTo" (e.g. ["bpmn:Task", "bpmn:ServiceTask"]).');
        }
        if (!Array.isArray(parsed.properties)) {
          errors.push('Template must contain a "properties" array.');
        } else {
          const seenBindings = new Set<string>();
          parsed.properties.forEach((p: any, idx: number) => {
            const bName = (p.binding?.name || '').trim();
            if (!bName) {
              errors.push(`Property #${idx + 1} ("${p.label || 'Unnamed'}") is missing binding.name.`);
            } else {
              if (seenBindings.has(bName)) {
                errors.push(`Duplicate property binding.name "${bName}" found.`);
              }
              seenBindings.add(bName);
            }
            if (p.editable === false && (p.value === undefined || String(p.value).trim() === '')) {
              errors.push(
                `Burned/Locked property "${p.label || bName}" (editable: false) must have a non-empty value.`
              );
            }
          });
        }
      }
    } catch (e: any) {
      errors.push(`Invalid JSON syntax: ${e?.message || 'unable to parse'}`);
    }

    this.validationErrors = errors;
    this.validationPassed = errors.length === 0;
    return this.validationPassed;
  }

  saveTemplate(closeAfterSave: boolean): void {
    if (!this.validateTemplate()) {
      return;
    }

    const name = (this.designerForm.get('name')?.value || '').trim();
    const rawCode = (this.designerForm.get('code')?.value || '').trim();
    const code = rawCode || this.toKebabCase(name);
    const description = (this.designerForm.get('description')?.value || '').trim();
    const activityFamily = (this.activityFamilyCtrl.value || 'default').trim();
    const vnamespace = (this.vnamespaceCtrl.value || 'default').trim();
    const parentTemplateId = (this.designerForm.get('parentTemplateId')?.value || '').trim();
    const rootActivity = (this.designerForm.get('rootActivity')?.value || '').trim();
    const isActive = !!this.designerForm.get('isActive')?.value;

    // Synchronize the top-level id and name inside the JSON payload with the template's code and name
    let finalPayloadStr = this.payloadJson;
    try {
      const parsed = JSON.parse(this.payloadJson);
      parsed.id = code;
      parsed.name = name;
      parsed.category = { id: activityFamily, name: `Family: ${activityFamily}` };
      finalPayloadStr = JSON.stringify(parsed, null, 2);
      this.payloadJson = finalPayloadStr;
    } catch {
      // Validated above
    }

    const payloadBody: Partial<ActivityTemplate> = {
      code,
      name,
      description,
      activityFamily,
      parentTemplateId,
      rootActivity,
      vnamespace,
      isActive,
      payload: finalPayloadStr
    };

    const req$ = this.isEditing
      ? this.scope === 'global'
        ? this.activityTemplatesService.updateGlobalTemplate(this.editingTemplateId, payloadBody)
        : this.activityTemplatesService.updateTenantTemplate(
            this.tenantCode,
            this.editingTemplateId,
            payloadBody
          )
      : this.scope === 'global'
        ? this.activityTemplatesService.createGlobalTemplate(payloadBody)
        : this.activityTemplatesService.createTenantTemplate(this.tenantCode, payloadBody);

    req$.subscribe({
      next: (saved: ActivityTemplate) => {
        this.successMsg = `Activity template "${saved.name}" saved successfully.`;
        if (!this.isEditing && saved.id) {
          this.isEditing = true;
          this.editingTemplateId = saved.id;
        }
        if (closeAfterSave) {
          this.showModal = false;
        }
        this.loadTemplates();
      },
      error: (err: any) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
      }
    });
  }

  openDetailModal(tpl: ActivityTemplate): void {
    this.selectedTemplate = tpl;
    this.detailPayloadText = this.activityTemplatesService.decodePayload(tpl.payload);
    this.showDetailModal = true;
  }

  openDeleteModal(tpl: ActivityTemplate): void {
    this.templateToDelete = tpl;
    this.showDeleteModal = true;
  }

  deleteTemplate(): void {
    if (!this.templateToDelete?.id) return;

    const req$ =
      this.scope === 'global'
        ? this.activityTemplatesService.deleteGlobalTemplate(this.templateToDelete.id)
        : this.activityTemplatesService.deleteTenantTemplate(
            this.tenantCode,
            this.templateToDelete.id
          );

    req$.subscribe({
      next: () => {
        this.successMsg = `Activity template "${this.templateToDelete?.name}" deleted.`;
        this.showDeleteModal = false;
        this.templateToDelete = null;
        this.loadTemplates();
      },
      error: (err: any) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.showDeleteModal = false;
      }
    });
  }

  nextPage(): void {
    if (this.nextCursor) {
      this.cursors.push(this.nextCursor);
      this.cursor = this.nextCursor;
      this.loadTemplates();
    }
  }

  prevPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.cursor = this.cursors[this.cursors.length - 1];
      this.loadTemplates();
    }
  }
}
