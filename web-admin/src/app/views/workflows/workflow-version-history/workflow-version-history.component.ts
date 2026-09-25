import { Component, OnInit, ViewChild } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import {
  TableModule,
  ButtonModule,
  CardModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ModalModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition, WorkflowDefinitionVersion } from '../services/workflows.service';
import { ErrorUtil } from '../../../shared/utils/error.util';
import { BpmnDesignerComponent, DEFAULT_BPMN_XML } from '../../../shared/components/bpmn-designer/bpmn-designer.component';

@Component({
  selector: 'app-workflow-version-history',
  templateUrl: './workflow-version-history.component.html',
  styleUrls: ['./workflow-version-history.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    TableModule,
    ButtonModule,
    CardModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    ModalModule,
    IconDirective,
    BpmnDesignerComponent
  ]
})
export class WorkflowVersionHistoryComponent implements OnInit {
  workflowId: string = '';
  scope: 'global' | 'tenant' = 'global';
  tenantCode: string = '';

  workflow: WorkflowDefinition | null = null;
  loadingWorkflow: boolean = false;

  versionHistoryList: WorkflowDefinitionVersion[] = [];
  loadingVersionHistory: boolean = false;
  versionHistoryError: string = '';

  showVersionDiagramModal: boolean = false;
  selectedVersionRecord: WorkflowDefinitionVersion | null = null;
  versionDiagramPayload: string = '';

  @ViewChild('versionBpmnDesigner') versionBpmnDesigner?: BpmnDesignerComponent;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private location: Location,
    private workflowsService: WorkflowsService
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
        this.loadVersionHistory();
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
          scope: this.scope
        };
        this.loadingWorkflow = false;
      },
      error: () => {
        this.loadingWorkflow = false;
      }
    });
  }

  loadVersionHistory(): void {
    if (!this.workflowId) return;
    this.loadingVersionHistory = true;
    this.versionHistoryError = '';

    const req$ = this.scope === 'global'
      ? this.workflowsService.getGlobalWorkflowVersions(this.workflowId)
      : this.workflowsService.getTenantWorkflowVersions(this.tenantCode, this.workflowId);

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

        if (this.versionHistoryList.length === 0 && this.workflow) {
          this.versionHistoryList = [{
            id: this.workflow.id || 'v1',
            workflowDefinitionId: this.workflow.id || '',
            version: this.workflow.version || 1,
            vnamespace: this.workflow.vnamespace || '',
            payload: this.workflow.payload || '',
            payloadFormat: this.workflow.payloadFormat || 'bpmn',
            structuralHash: 'Current (Initial Version)',
            createdAt: this.workflow.updatedAt || this.workflow.createdAt || new Date().toISOString()
          }];
        } else if (this.workflow && this.versionHistoryList.length > 0) {
          const minVersion = Math.min(...this.versionHistoryList.map(v => v.version));
          if (minVersion > 1) {
            this.versionHistoryList.push({
              id: `${this.workflow.id}-v1`,
              workflowDefinitionId: this.workflow.id || '',
              version: 1,
              vnamespace: this.workflow.vnamespace || '',
              payload: this.workflow.payload || '',
              payloadFormat: this.workflow.payloadFormat || 'bpmn',
              structuralHash: 'Current (Initial Version)',
              createdAt: this.workflow.createdAt || new Date().toISOString()
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

  navigateBack(): void {
    if (this.tenantCode) {
      this.router.navigate(['/tenants', this.tenantCode, 'management'], { queryParams: { tab: 'workflows' } });
    } else {
      this.router.navigate(['/workflows']);
    }
  }
}
