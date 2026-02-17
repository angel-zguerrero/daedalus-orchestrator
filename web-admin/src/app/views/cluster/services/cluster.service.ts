import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Member {
  ip: string;
  port: number;
}

export interface NodeInfo {
  replica_id: number;
  shard_id: number;
  self_member: Member;
  roles: string[];
}

export interface TenantNodeInfo extends NodeInfo {
  tenant_id?: string;
}

// New NodeSchedulerBalancingState interface
export interface NodeSchedulerBalancingState {
  ID: string;
  BalancingId: string;
  Status: string;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface DisplayNode extends NodeInfo {
  node_type: string;
  tenant_id?: string;
}

// New interfaces for cluster config information
export interface ClusterNodeInfo {
  node_id: number;
  address: string;
  shard_id: number;
}

export interface ClusterConfigInfo {
  shard_id: number;
  nodes: ClusterNodeInfo[];
  total: number;
}

export interface NodeConfiguration {
  master_node: NodeInfo;
  tenant_nodes: TenantNodeInfo[];
}

export interface ClusterPortInfo {
  cluster_base_port: number;
  rest_port: number;
  grpc_port: number;
}

export interface EnhancedClusterInfo {
  cluster_config: ClusterConfigInfo[];
  node_configuration: NodeConfiguration;
  balancing_state?: NodeSchedulerBalancingState;
  port_configuration: ClusterPortInfo;
}

// Legacy interface for backward compatibility
export interface ClusterInfo {
  master_node: NodeInfo;
  tenant_nodes: TenantNodeInfo[];
}

export interface AddReplicaRequest {
  replica_id: number;
  host: string;
  port: number;
  shard_ids?: number[];
}

export interface ShardResult {
  shard_id: number;
  success: boolean;
  message: string;
  error?: string;
}

export interface AddReplicaResponse {
  success: boolean;
  message: string;
  replica_id: number;
  results: { [key: string]: ShardResult };
}

@Injectable({
  providedIn: 'root'
})
export class ClusterService {
  private apiUrl = '/rest-api/v1/cluster';

  constructor(private http: HttpClient) { }

  /**
   * Get enhanced cluster information including cluster config and node configuration
   */
  getEnhancedClusterInfo(): Observable<EnhancedClusterInfo> {
    return this.http.get<EnhancedClusterInfo>(`${this.apiUrl}/info`);
  }

  /**
   * Get cluster information including master node and tenant nodes (legacy method for backward compatibility)
   */
  getClusterInfo(): Observable<ClusterInfo> {
    return this.http.get<ClusterInfo>(`${this.apiUrl}/info`);
  }

  /**
   * Add a new replica to the cluster
   */
  addReplica(request: AddReplicaRequest): Observable<AddReplicaResponse> {
    return this.http.post<AddReplicaResponse>(`${this.apiUrl}/replicas`, request);
  }
}