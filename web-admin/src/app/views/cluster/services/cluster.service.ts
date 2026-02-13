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

export interface DisplayNode extends NodeInfo {
  node_type: string;
  tenant_id?: string;
}

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
   * Get cluster information including master node and tenant nodes
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