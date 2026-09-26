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
  ModalModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { WorkflowsService, WorkflowDefinition, WorkflowExecution, WorkflowExecutionDetail } from '../services/workflows.service';
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
  loadingExecutionDetail: boolean = false;
  executionDetailError: string = '';
  activeDetailTab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'diagram' = 'overview';

  @ViewChild('executionBpmnDesigner') executionBpmnDesigner?: BpmnDesignerComponent;

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
          scope: this.scope
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

  openExecutionDetail(executionId: string): void {
    this.selectedExecutionId = executionId;
    this.showExecutionDetailModal = true;
    this.loadingExecutionDetail = true;
    this.executionDetailError = '';
    this.selectedExecutionDetail = null;

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

  selectDetailTab(tab: 'overview' | 'payloads' | 'activities' | 'tokens' | 'diagram'): void {
    this.activeDetailTab = tab;
    if (tab === 'diagram') {
      setTimeout(() => this.highlightExecutionDiagram(), 250);
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
