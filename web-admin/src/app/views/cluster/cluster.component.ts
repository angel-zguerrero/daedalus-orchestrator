import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  ClusterService, 
  EnhancedClusterInfo, 
  ClusterConfigInfo, 
  ClusterNodeInfo,
  NodeInfo, 
  TenantNodeInfo, 
  AddReplicaRequest, 
  DisplayNode 
} from './services/cluster.service';
import { TableModule, UtilitiesModule, ButtonModule, ModalModule, CardModule, FormModule, GridModule, AlertComponent, SpinnerComponent, BadgeComponent } from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { ErrorUtil } from '../../shared/utils/error.util';

@Component({
  selector: 'app-cluster',
  templateUrl: './cluster.component.html',
  styleUrls: ['./cluster.component.scss'],
  standalone: true,
  imports: [
    AlertComponent,
    CommonModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    ReactiveFormsModule,
    FormsModule,
    SpinnerComponent,
    BadgeComponent,
    IconDirective
  ]
})
export class ClusterComponent implements OnInit {
  // Primary cluster information from GetClusterConfig
  enhancedClusterInfo: EnhancedClusterInfo | null = null;
  clusterConfigData: ClusterConfigInfo[] = [];
  totalClusterNodes = 0;
  totalShards = 0;

  // Secondary information: legacy node configuration for compatibility
  allNodes: DisplayNode[] = [];

  public addNodeModalVisible = false;
  public showAlert = false;
  public alertType = 'danger';
  public alertMessage = '';
  public loading = false;
  public addingNode = false;

  addNodeForm: FormGroup;

  constructor(
    private clusterService: ClusterService,
    private formBuilder: FormBuilder
  ) {
    this.addNodeForm = this.formBuilder.group({
      replica_id: ['', [Validators.required, Validators.min(1)]],
      host: ['127.0.0.1', [Validators.required, Validators.pattern(/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/)]],
      port: ['', [Validators.required, Validators.min(1024), Validators.max(65535)]],
      specific_shards: [false],
      shard_ids: [[]]
    });
  }

  ngOnInit(): void {
    this.loadClusterInfo();
  }

  loadClusterInfo(): void {
    this.loading = true;
    this.showAlert = false;

    this.clusterService.getEnhancedClusterInfo().subscribe({
      next: (enhancedClusterInfo: EnhancedClusterInfo) => {
        this.enhancedClusterInfo = enhancedClusterInfo;
        this.clusterConfigData = enhancedClusterInfo.cluster_config;
        this.calculateClusterStats();
        this.buildAllNodesList(); // For backward compatibility with secondary info
        this.loading = false;
      },
      error: (error) => {
        this.handleError('Failed to load cluster information', error);
        this.loading = false;
      }
    });
  }

  calculateClusterStats(): void {
    this.totalClusterNodes = 0;
    this.totalShards = this.clusterConfigData.length;
    
    this.clusterConfigData.forEach(shard => {
      this.totalClusterNodes += shard.total;
    });
  }

  buildAllNodesList(): void {
    if (!this.enhancedClusterInfo || !this.enhancedClusterInfo.node_configuration) return;

    this.allNodes = [];

    const nodeConfig = this.enhancedClusterInfo.node_configuration;

    // Add master node if it exists
    if (nodeConfig.master_node) {
      this.allNodes.push({
        ...nodeConfig.master_node,
        node_type: 'Master'
      } as DisplayNode);
    }

    // Add tenant nodes if they exist
    if (nodeConfig.tenant_nodes && Array.isArray(nodeConfig.tenant_nodes)) {
      nodeConfig.tenant_nodes.forEach(tenantNode => {
        this.allNodes.push({
          ...tenantNode,
          node_type: 'Tenant'
        } as DisplayNode);
      });
    }

    // Sort by replica_id
    this.allNodes.sort((a, b) => a.replica_id - b.replica_id);
  }

  openAddNodeModal(): void {
    this.addNodeForm.reset({
      replica_id: '',
      host: '127.0.0.1',
      port: '',
      specific_shards: false,
      shard_ids: []
    });
    this.addNodeModalVisible = true;
  }

  closeAddNodeModal(): void {
    this.addNodeModalVisible = false;
    this.addNodeForm.reset();
  }

  onAddNode(): void {
    if (this.addNodeForm.invalid) {
      this.markFormGroupTouched(this.addNodeForm);
      return;
    }

    this.addingNode = true;
    this.showAlert = false;

    const formValue = this.addNodeForm.value;
    const request: AddReplicaRequest = {
      replica_id: parseInt(formValue.replica_id),
      host: formValue.host,
      port: parseInt(formValue.port)
    };

    // Only include shard_ids if specific shards are selected
    if (formValue.specific_shards && formValue.shard_ids && formValue.shard_ids.length > 0) {
      request.shard_ids = formValue.shard_ids.map((id: string) => parseInt(id));
    }

    this.clusterService.addReplica(request).subscribe({
      next: (response) => {
        this.addingNode = false;
        this.closeAddNodeModal();

        if (response.success) {
          this.showSuccessAlert(`Node ${request.replica_id} added successfully to the cluster`);
          this.loadClusterInfo(); // Reload cluster info
        } else {
          this.showErrorAlert(`Failed to add node: ${response.message}`);
        }
      },
      error: (error) => {
        this.addingNode = false;
        this.handleError('Failed to add node to cluster', error);
      }
    });
  }

  getAvailableShards(): number[] {
    if (!this.enhancedClusterInfo || !this.enhancedClusterInfo.node_configuration) return [];

    const nodeConfig = this.enhancedClusterInfo.node_configuration;
    const shards: number[] = [];
    
    // Add master node shard if it exists
    if (nodeConfig.master_node && nodeConfig.master_node.shard_id !== undefined) {
      shards.push(nodeConfig.master_node.shard_id);
    }
    
    // Add tenant nodes shards if they exist
    if (nodeConfig.tenant_nodes && Array.isArray(nodeConfig.tenant_nodes)) {
      nodeConfig.tenant_nodes.forEach(node => {
        if (node && node.shard_id !== undefined && !shards.includes(node.shard_id)) {
          shards.push(node.shard_id);
        }
      });
    }

    return shards.sort((a, b) => a - b);
  }

  getClusterNodesByShardId(shardId: number): ClusterNodeInfo[] {
    const shard = this.clusterConfigData.find(s => s.shard_id === shardId);
    return shard ? shard.nodes : [];
  }

  getShardDisplayName(shardId: number): string {
    if (!this.enhancedClusterInfo || !this.enhancedClusterInfo.node_configuration) {
      return `Shard ${shardId}`;
    }
    
    const nodeConfig = this.enhancedClusterInfo.node_configuration;
    
    // Check if it's the master shard
    if (nodeConfig.master_node && nodeConfig.master_node.shard_id === shardId) {
      return `Master Shard (${shardId})`;
    }
    
    // Check if it's a tenant shard
    if (nodeConfig.tenant_nodes && Array.isArray(nodeConfig.tenant_nodes)) {
      const tenantNode = nodeConfig.tenant_nodes.find(node => node && node.shard_id === shardId);
      if (tenantNode && tenantNode.tenant_id) {
        return `Tenant Shard ${shardId} (${tenantNode.tenant_id})`;
      }
    }
    
    return `Shard ${shardId}`;
  }

  getNodeTypeColor(nodeType: string): string {
    switch (nodeType) {
      case 'Master':
        return 'primary';
      case 'Tenant':
        return 'success';
      default:
        return 'secondary';
    }
  }

  getRolesDisplay(roles: string[]): string {
    return roles.join(', ');
  }

  private markFormGroupTouched(formGroup: FormGroup): void {
    Object.keys(formGroup.controls).forEach(key => {
      const control = formGroup.get(key);
      control?.markAsTouched();
    });
  }

  private handleError(message: string, error: any): void {
    console.error(message, error);
    const errorMessage = ErrorUtil.formatErrorMessage(error);
    this.showErrorAlert(`${message}: ${errorMessage}`);
  }

  private showErrorAlert(message: string): void {
    this.alertType = 'danger';
    this.alertMessage = message;
    this.showAlert = true;
  }

  private showSuccessAlert(message: string): void {
    this.alertType = 'success';
    this.alertMessage = message;
    this.showAlert = true;
  }

  refreshClusterInfo(): void {
    this.loadClusterInfo();
  }

  onShardSelectionChange(event: any, shardId: number): void {
    const currentShardIds = this.addNodeForm.get('shard_ids')?.value || [];

    if (event.target.checked) {
      if (!currentShardIds.includes(shardId)) {
        currentShardIds.push(shardId);
      }
    } else {
      const index = currentShardIds.indexOf(shardId);
      if (index > -1) {
        currentShardIds.splice(index, 1);
      }
    }

    this.addNodeForm.get('shard_ids')?.setValue(currentShardIds);
  }
}