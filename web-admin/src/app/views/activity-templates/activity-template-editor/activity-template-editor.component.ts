import { Component, OnInit } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { map, switchMap, startWith, debounceTime, catchError } from 'rxjs/operators';
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
import { ActivityTemplatesService, ActivityTemplate } from '../services/activity-templates.service';
import { VNamespacesService } from '../../tenants/tenant-management/services/vnamespaces.service';
import { ErrorUtil } from '../../../shared/utils/error.util';
import elementTemplatesData from '../../../shared/components/bpmn-designer/element-templates.json';

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
  selector: 'app-activity-template-editor',
  templateUrl: './activity-template-editor.component.html',
  styleUrls: ['./activity-template-editor.component.scss'],
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
    BadgeComponent,
    ModalModule,
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule
  ]
})
export class ActivityTemplateEditorComponent implements OnInit {
  isEditing: boolean = false;
  editingTemplateId: string = '';
  scope: 'global' | 'tenant' = 'global';
  tenantCode: string = '';
  currentStep: 'identity' | 'designer' = 'identity';

  loading: boolean = false;
  saving: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  showAdvancedSettings: boolean = false;
  identityForm: FormGroup;
  designerForm: FormGroup;

  // Activity Family Autocomplete
  activityFamilyCtrl = new FormControl('default', Validators.required);
  filteredActivityFamilies!: Observable<string[]>;
  knownFamilies: string[] = ['default', 'cache', 'http', 'logging', 'messaging', 'database'];

  // VNamespace Autocomplete
  vnamespaceCtrl = new FormControl('default');
  filteredVNamespaces!: Observable<any[]>;

  // Parent Templates
  builtinOptions: ParentTemplateOption[] = [];
  customParentOptions: ParentTemplateOption[] = [];
  selectedParentOption: ParentTemplateOption | null = null;

  // Designer Payload & Live Preview
  payloadJson: string = '';
  previewGroups: PreviewGroup[] = [];
  validationErrors: string[] = [];
  validationPassed: boolean = false;

  constructor(
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router,
    private location: Location,
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

    this.filteredActivityFamilies = this.activityFamilyCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(200),
      map((val) => this._filterFamilies(val || ''))
    );

    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap((value) => this._filterVNamespaces(value || ''))
    );

    // Synchronize identityForm and designerForm controls
    this.identityForm.valueChanges.subscribe((val) => {
      this.designerForm.patchValue({
        name: val.name,
        code: val.code,
        activityFamily: val.activityFamily,
        vnamespace: val.vnamespace,
        parentTemplateId: val.parentTemplateId
      }, { emitEvent: false });

      if (val.parentTemplateId) {
        this.onParentSelectionChange(val.parentTemplateId);
      }
      this.onDesignerNameOrCodeChange();
    });

    this.designerForm.valueChanges.subscribe((val) => {
      this.identityForm.patchValue({
        name: val.name,
        code: val.code,
        activityFamily: val.activityFamily,
        vnamespace: val.vnamespace,
        parentTemplateId: val.parentTemplateId
      }, { emitEvent: false });
    });
  }

  ngOnInit(): void {
    const qParams = this.route.snapshot.queryParams;
    if (qParams['tenantCode']) {
      this.scope = 'tenant';
      this.tenantCode = qParams['tenantCode'];
    }

    const templateId = this.route.snapshot.params['id'];
    this.loadCustomParents(templateId);
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

  loadCustomParents(templateId?: string): void {
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

          if (templateId) {
            this.isEditing = true;
            this.editingTemplateId = templateId;
            this.currentStep = 'designer';
            this.loadTemplate(templateId);
          } else {
            this.isEditing = false;
            this.currentStep = 'identity';
            this.initCreateMode();
          }
        },
        error: () => {
          if (templateId) {
            this.isEditing = true;
            this.editingTemplateId = templateId;
            this.currentStep = 'designer';
            this.loadTemplate(templateId);
          } else {
            this.isEditing = false;
            this.currentStep = 'identity';
            this.initCreateMode();
          }
        }
      });
  }

  initCreateMode(): void {
    this.activityFamilyCtrl.setValue('default', { emitEvent: false });
    this.vnamespaceCtrl.setValue('default', { emitEvent: false });

    const defaultParent = this.builtinOptions[1] || this.builtinOptions[0];
    this.selectedParentOption = defaultParent || null;

    this.identityForm.patchValue({
      name: '',
      code: '',
      activityFamily: 'default',
      vnamespace: 'default',
      parentTemplateId: defaultParent ? defaultParent.id : ''
    }, { emitEvent: false });

    this.designerForm.patchValue({
      name: '',
      code: '',
      description: '',
      activityFamily: 'default',
      vnamespace: 'default',
      parentTemplateId: defaultParent ? defaultParent.id : '',
      rootActivity: defaultParent ? defaultParent.id : '',
      isActive: true
    }, { emitEvent: false });
  }

  loadTemplate(id: string): void {
    this.loading = true;
    const obs =
      this.scope === 'global'
        ? this.activityTemplatesService.getGlobalTemplate(id)
        : this.activityTemplatesService.getTenantTemplate(this.tenantCode, id);

    obs.subscribe({
      next: (tpl: ActivityTemplate) => {
        this.activityFamilyCtrl.setValue(tpl.activityFamily || 'default', { emitEvent: false });
        this.vnamespaceCtrl.setValue(tpl.vnamespace || 'default', { emitEvent: false });

        this.identityForm.patchValue({
          name: tpl.name,
          code: tpl.code,
          activityFamily: tpl.activityFamily || 'default',
          vnamespace: tpl.vnamespace || 'default',
          parentTemplateId: tpl.parentTemplateId
        }, { emitEvent: false });

        this.designerForm.patchValue({
          name: tpl.name,
          code: tpl.code,
          description: tpl.description || '',
          activityFamily: tpl.activityFamily || 'default',
          vnamespace: tpl.vnamespace || 'default',
          parentTemplateId: tpl.parentTemplateId,
          rootActivity: tpl.rootActivity || tpl.parentTemplateId,
          isActive: tpl.isActive
        }, { emitEvent: false });

        this.onParentSelectionChange(tpl.parentTemplateId);

        this.payloadJson = this.activityTemplatesService.decodePayload(tpl.payload);
        this.parsePreviewGroups();
        this.validateTemplate();
        this.loading = false;
      },
      error: (err: any) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
        this.loading = false;
      }
    });
  }

  onParentSelectionChange(parentId: string): void {
    const allOptions = [...this.builtinOptions, ...this.customParentOptions];
    this.selectedParentOption = allOptions.find((o) => o.id === parentId) || null;
  }

  proceedToDesigner(): void {
    const name = this.identityForm.get('name')?.value?.trim();
    if (!name) return;
    const parentId = this.identityForm.get('parentTemplateId')?.value;
    this.onParentSelectionChange(parentId);
    if (!this.selectedParentOption) return;

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
    }, { emitEvent: false });

    if (!this.payloadJson || !this.isEditing) {
      const baseJson = JSON.parse(JSON.stringify(this.selectedParentOption.rawTemplate));
      baseJson.id = code;
      baseJson.name = name;
      baseJson.description = `Custom ${this.selectedParentOption.name} template (${family})`;
      baseJson.category = {
        id: family,
        name: `Family: ${family}`
      };

      if (Array.isArray(baseJson.properties)) {
        baseJson.properties = baseJson.properties.map((p: any) => ({
          ...p,
          editable: p.editable !== undefined ? p.editable : true
        }));
      }

      this.payloadJson = JSON.stringify(baseJson, null, 2);
    }
    this.parsePreviewGroups();
    this.validateTemplate();
    this.currentStep = 'designer';
  }

  setStep(step: 'identity' | 'designer'): void {
    if (step === 'designer' && this.currentStep === 'identity' && !this.isEditing && !this.payloadJson) {
      this.proceedToDesigner();
    } else {
      this.currentStep = step;
    }
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
    } catch {}
  }

  onPayloadChange(newJson: string): void {
    this.payloadJson = newJson;
    this.parsePreviewGroups();
    this.validateTemplate();
  }

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
    } catch {}
  }

  onPreviewPropertyValueChange(prop: PreviewProperty, newValue: any): void {
    try {
      const parsed = JSON.parse(this.payloadJson);
      if (Array.isArray(parsed.properties) && parsed.properties[prop.index]) {
        parsed.properties[prop.index].value = newValue;
        this.payloadJson = JSON.stringify(parsed, null, 2);
        prop.value = newValue;
        this.validateTemplate();
      }
    } catch {}
  }

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
    } catch {}
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
    if (!this.validateTemplate()) return;
    this.saving = true;
    this.showAlert = false;
    this.successMsg = '';

    const name = (this.designerForm.get('name')?.value || '').trim();
    const rawCode = (this.designerForm.get('code')?.value || '').trim();
    const code = rawCode || this.toKebabCase(name);
    const description = (this.designerForm.get('description')?.value || '').trim();
    const activityFamily = (this.activityFamilyCtrl.value || 'default').trim();
    const vnamespace = (this.vnamespaceCtrl.value || 'default').trim();
    const parentTemplateId = (this.designerForm.get('parentTemplateId')?.value || '').trim();
    const rootActivity = (this.designerForm.get('rootActivity')?.value || '').trim();
    const isActive = !!this.designerForm.get('isActive')?.value;

    let finalPayloadStr = this.payloadJson;
    try {
      const parsed = JSON.parse(this.payloadJson);
      parsed.id = code;
      parsed.name = name;
      parsed.category = { id: activityFamily, name: `Family: ${activityFamily}` };
      finalPayloadStr = JSON.stringify(parsed, null, 2);
      this.payloadJson = finalPayloadStr;
    } catch {}

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
        : this.activityTemplatesService.updateTenantTemplate(this.tenantCode, this.editingTemplateId, payloadBody)
      : this.scope === 'global'
        ? this.activityTemplatesService.createGlobalTemplate(payloadBody)
        : this.activityTemplatesService.createTenantTemplate(this.tenantCode, payloadBody);

    req$.subscribe({
      next: (saved: ActivityTemplate) => {
        this.saving = false;
        this.successMsg = `Activity template "${saved.name}" saved successfully.`;
        if (!this.isEditing && saved.id) {
          this.isEditing = true;
          this.editingTemplateId = saved.id;
          this.location.replaceState(
            this.scope === 'tenant'
              ? `/activity-templates/${saved.id}/edit?tenantCode=${this.tenantCode}`
              : `/activity-templates/${saved.id}/edit`
          );
        }
        if (closeAfterSave) {
          setTimeout(() => {
            this.goBack();
          }, 800);
        } else {
          setTimeout(() => {
            this.successMsg = '';
          }, 4000);
        }
      },
      error: (err: any) => {
        this.saving = false;
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
      }
    });
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

  goBack(): void {
    const queryParams = this.scope === 'tenant' ? { tenantCode: this.tenantCode } : {};
    this.router.navigate(['/activity-templates'], { queryParams });
  }
}
