import { Component, OnInit, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
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
  TabsModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { ConfigsSecretsService, EnvGroup, EnvVar } from './services/configs-secrets.service';
import { ErrorUtil } from '../../shared/utils/error.util';

@Component({
  selector: 'app-configs-secrets',
  templateUrl: './configs-secrets.component.html',
  styleUrls: ['./configs-secrets.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
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
    IconDirective
  ]
})
export class ConfigsSecretsComponent implements OnInit, OnChanges {
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';

  groups: EnvGroup[] = [];
  filteredGroups: EnvGroup[] = [];
  loading: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';
  successMsg: string = '';

  searchQuery: string = '';
  selectedTypeFilter: 'all' | 'config' | 'secret' = 'all';

  // Pagination
  cursor: string = '';
  nextCursor: string = '';
  cursors: string[] = [''];
  pageSize: number = 20;

  // Group Modal
  showGroupModal: boolean = false;
  groupForm: FormGroup;
  isEditingGroup: boolean = false;
  editingGroupId: string = '';

  // Manage Vars Modal / Detail
  showVarsModal: boolean = false;
  selectedGroup: EnvGroup | null = null;
  vars: EnvVar[] = [];
  varsLoading: boolean = false;
  varsError: string = '';

  // Single Var Form
  varForm: FormGroup;
  isEditingVar: boolean = false;
  editingVarId: string = '';

  // Bulk Edit / .env Raw Text Mode
  isBulkMode: boolean = false;
  rawEnvText: string = '';

  // Delete Confirm Modal
  showDeleteModal: boolean = false;
  groupToDelete: EnvGroup | null = null;

  constructor(
    private fb: FormBuilder,
    private configService: ConfigsSecretsService
  ) {
    this.groupForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', [Validators.pattern('^[A-Za-z0-9_-]*$')]],
      description: [''],
      type: ['config', Validators.required]
    });

    this.varForm = this.fb.group({
      key: ['', [Validators.required, Validators.pattern('^[A-Za-z0-9_.-]+$')]],
      value: [''],
      description: ['']
    });
  }

  ngOnInit(): void {
    this.loadGroups();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['tenantCode'] || changes['scope']) {
      this.cursor = '';
      this.cursors = [''];
      this.loadGroups();
    }
  }

  private normalizeGroup(g: any): EnvGroup {
    return {
      id: g.id || g.ID || '',
      code: g.code || g.Code || '',
      name: g.name || g.Name || g.code || g.Code || '',
      description: g.description || g.Description || '',
      type: (g.type || g.Type || 'config').toLowerCase() as 'config' | 'secret',
      scope: (g.scope || g.Scope || this.scope).toLowerCase() as 'global' | 'tenant',
      tenantID: g.tenantID || g.TenantID || '',
      vnamespace: g.vnamespace || g.VNamespace || '',
      createdAt: g.createdAt || g.CreatedAt || '',
      updatedAt: g.updatedAt || g.UpdatedAt || ''
    };
  }

  private normalizeVar(v: any): EnvVar {
    return {
      id: v.id || v.ID || '',
      groupID: v.groupID || v.GroupID || '',
      key: v.key || v.Key || '',
      value: v.value !== undefined ? v.value : (v.Value !== undefined ? v.Value : ''),
      description: v.description || v.Description || '',
      createdAt: v.createdAt || v.CreatedAt || '',
      updatedAt: v.updatedAt || v.UpdatedAt || '',
      showSecret: false
    };
  }

  loadGroups(): void {
    this.loading = true;
    this.showAlert = false;
    this.errorMessage = '';

    const req$ = this.scope === 'global'
      ? this.configService.getGlobalGroups(this.pageSize, this.cursor)
      : this.configService.getTenantGroups(this.tenantCode, this.pageSize, this.cursor);

    req$.subscribe({
      next: (res) => {
        const rawEntities = res.Entities || res.entities || [];
        this.groups = rawEntities.map((g: any) => this.normalizeGroup(g));
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
    let result = [...this.groups];

    if (this.selectedTypeFilter !== 'all') {
      result = result.filter(g => g.type === this.selectedTypeFilter);
    }

    if (this.searchQuery.trim()) {
      const q = this.searchQuery.toLowerCase().trim();
      result = result.filter(g =>
        g.code.toLowerCase().includes(q) ||
        (g.name && g.name.toLowerCase().includes(q)) ||
        (g.description && g.description.toLowerCase().includes(q))
      );
    }

    this.filteredGroups = result;
  }

  nextPage(): void {
    if (this.nextCursor) {
      this.cursors.push(this.nextCursor);
      this.cursor = this.nextCursor;
      this.loadGroups();
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.cursor = this.cursors[this.cursors.length - 1];
      this.loadGroups();
    }
  }

  openCreateGroupModal(): void {
    this.isEditingGroup = false;
    this.editingGroupId = '';
    this.groupForm.reset({
      type: 'config'
    });
    this.groupForm.get('code')?.enable();
    this.groupForm.get('type')?.enable();
    this.showGroupModal = true;
  }

  openEditGroupModal(group: EnvGroup): void {
    this.isEditingGroup = true;
    this.editingGroupId = group.id || '';
    this.groupForm.patchValue({
      code: group.code,
      name: group.name,
      description: group.description,
      type: group.type
    });
    this.groupForm.get('code')?.disable();
    this.groupForm.get('type')?.disable();
    this.showGroupModal = true;
  }

  saveGroup(): void {
    if (this.groupForm.invalid) return;

    const val = this.groupForm.getRawValue();
    const payload: Partial<EnvGroup> = {
      code: val.code ? val.code.trim() : '',
      name: val.name ? val.name.trim() : '',
      description: val.description,
      type: val.type,
      scope: this.scope
    };

    if (this.isEditingGroup) {
      const update$ = this.scope === 'global'
        ? this.configService.updateGlobalGroup(this.editingGroupId, payload)
        : this.configService.updateTenantGroup(this.tenantCode, this.editingGroupId, payload);

      update$.subscribe({
        next: () => {
          this.showGroupModal = false;
          this.successMsg = 'Group updated successfully.';
          this.loadGroups();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    } else {
      const create$ = this.scope === 'global'
        ? this.configService.createGlobalGroup(payload)
        : this.configService.createTenantGroup(this.tenantCode, payload);

      create$.subscribe({
        next: () => {
          this.showGroupModal = false;
          this.successMsg = 'Group created successfully.';
          this.loadGroups();
          setTimeout(() => this.successMsg = '', 3000);
        },
        error: (err) => {
          this.errorMessage = ErrorUtil.formatErrorMessage(err);
          this.showAlert = true;
        }
      });
    }
  }

  confirmDeleteGroup(group: EnvGroup): void {
    this.groupToDelete = group;
    this.showDeleteModal = true;
  }

  deleteGroup(): void {
    if (!this.groupToDelete || !this.groupToDelete.id) return;

    const del$ = this.scope === 'global'
      ? this.configService.deleteGlobalGroup(this.groupToDelete.id)
      : this.configService.deleteTenantGroup(this.tenantCode, this.groupToDelete.id);

    del$.subscribe({
      next: () => {
        this.showDeleteModal = false;
        this.groupToDelete = null;
        this.successMsg = 'Group deleted successfully.';
        this.loadGroups();
        setTimeout(() => this.successMsg = '', 3000);
      },
      error: (err) => {
        this.errorMessage = ErrorUtil.formatErrorMessage(err);
        this.showAlert = true;
      }
    });
  }

  // --- VARIABLES MANAGEMENT ---

  openManageVarsModal(group: EnvGroup): void {
    this.selectedGroup = this.normalizeGroup(group);
    this.isBulkMode = false;
    this.varsError = '';
    this.showVarsModal = true;
    this.resetVarForm();
    this.loadVars();
  }

  loadVars(): void {
    const groupId = this.selectedGroup?.id;
    if (!groupId) return;

    this.varsLoading = true;
    this.varsError = '';

    const req$ = this.scope === 'global'
      ? this.configService.getGlobalVars(groupId)
      : this.configService.getTenantVars(this.tenantCode, groupId);

    req$.subscribe({
      next: (res) => {
        const rawEntities = res.Entities || res.entities || [];
        this.vars = rawEntities.map((v: any) => this.normalizeVar(v));
        this.varsLoading = false;
        this.prepareRawEnvText();
      },
      error: (err) => {
        this.varsError = ErrorUtil.formatErrorMessage(err);
        this.varsLoading = false;
      }
    });
  }

  toggleSecretVisibility(envVar: EnvVar): void {
    envVar.showSecret = !envVar.showSecret;
  }

  resetVarForm(): void {
    this.isEditingVar = false;
    this.editingVarId = '';
    this.varForm.reset();
  }

  editVar(v: EnvVar): void {
    this.isEditingVar = true;
    this.editingVarId = v.id || '';
    this.varForm.patchValue({
      key: v.key,
      value: v.value,
      description: v.description
    });
  }

  saveVar(): void {
    const groupId = this.selectedGroup?.id;
    if (this.varForm.invalid || !groupId) return;

    const val = this.varForm.value;
    const payload: Partial<EnvVar> = {
      id: this.editingVarId || undefined,
      key: val.key,
      value: val.value || '',
      description: val.description || ''
    };

    const save$ = this.scope === 'global'
      ? this.configService.saveGlobalVar(groupId, payload)
      : this.configService.saveTenantVar(this.tenantCode, groupId, payload);

    save$.subscribe({
      next: () => {
        this.resetVarForm();
        this.loadVars();
      },
      error: (err) => {
        this.varsError = ErrorUtil.formatErrorMessage(err);
      }
    });
  }

  deleteVar(v: EnvVar): void {
    const groupId = this.selectedGroup?.id;
    if (!v.id || !groupId) return;

    const del$ = this.scope === 'global'
      ? this.configService.deleteGlobalVar(groupId, v.id)
      : this.configService.deleteTenantVar(this.tenantCode, groupId, v.id);

    del$.subscribe({
      next: () => {
        this.loadVars();
      },
      error: (err) => {
        this.varsError = ErrorUtil.formatErrorMessage(err);
      }
    });
  }

  // --- RAW BULK .ENV MODE ---

  toggleBulkMode(): void {
    this.isBulkMode = !this.isBulkMode;
    if (this.isBulkMode) {
      this.prepareRawEnvText();
    }
  }

  prepareRawEnvText(): void {
    this.rawEnvText = this.vars
      .map(v => {
        const desc = v.description && v.description.trim() ? `# ${v.description.trim()}\n` : '';
        return `${desc}${v.key}=${v.value || ''}`;
      })
      .join('\n\n');
  }

  saveBulkEnv(): void {
    const groupId = this.selectedGroup?.id;
    if (!groupId) return;

    const lines = this.rawEnvText.split('\n');
    const parsedVars: EnvVar[] = [];
    let pendingDescription = '';

    for (let line of lines) {
      line = line.trim();
      if (!line) {
        pendingDescription = '';
        continue;
      }

      // Check if line is a comment line starting with # or --
      if (line.startsWith('#') || line.startsWith('--')) {
        let commentText = line.startsWith('#')
          ? line.substring(1).trim()
          : line.substring(2).trim();

        if (commentText.toLowerCase().startsWith('description:')) {
          commentText = commentText.substring(12).trim();
        }

        pendingDescription = commentText;
        continue;
      }

      // Check for KEY=VALUE pair
      const eqIdx = line.indexOf('=');
      if (eqIdx > 0) {
        const key = line.substring(0, eqIdx).trim();
        let rawValAndDesc = line.substring(eqIdx + 1).trim();
        let inlineDesc = '';

        // Check for inline comment (# or --)
        const hashCommentIdx = rawValAndDesc.indexOf(' #');
        const dashCommentIdx = rawValAndDesc.indexOf(' --');

        let commentIdx = -1;
        if (hashCommentIdx !== -1 && dashCommentIdx !== -1) {
          commentIdx = Math.min(hashCommentIdx, dashCommentIdx);
        } else if (hashCommentIdx !== -1) {
          commentIdx = hashCommentIdx;
        } else if (dashCommentIdx !== -1) {
          commentIdx = dashCommentIdx;
        }

        if (commentIdx !== -1) {
          inlineDesc = rawValAndDesc.substring(commentIdx + 2).trim();
          if (inlineDesc.toLowerCase().startsWith('description:')) {
            inlineDesc = inlineDesc.substring(12).trim();
          }
          rawValAndDesc = rawValAndDesc.substring(0, commentIdx).trim();
        }

        if ((rawValAndDesc.startsWith('"') && rawValAndDesc.endsWith('"')) ||
            (rawValAndDesc.startsWith("'") && rawValAndDesc.endsWith("'"))) {
          rawValAndDesc = rawValAndDesc.substring(1, rawValAndDesc.length - 1);
        }

        const finalDescription = inlineDesc || pendingDescription;
        const existingVar = this.vars.find(v => v.key.toLowerCase() === key.toLowerCase());

        if (key) {
          parsedVars.push({
            id: existingVar?.id || '',
            groupID: groupId,
            key: key,
            value: rawValAndDesc,
            description: finalDescription
          });
        }

        pendingDescription = '';
      }
    }

    const bulk$ = this.scope === 'global'
      ? this.configService.bulkSaveGlobalVars(groupId, parsedVars)
      : this.configService.bulkSaveTenantVars(this.tenantCode, groupId, parsedVars);

    bulk$.subscribe({
      next: () => {
        this.isBulkMode = false;
        this.loadVars();
      },
      error: (err) => {
        this.varsError = ErrorUtil.formatErrorMessage(err);
      }
    });
  }
}
