// package: tenant
// file: tenant.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as tenant_pb from "./tenant_pb";

interface ITenantServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    getTenantInfo: ITenantServiceService_IGetTenantInfo;
    getTenantSummary: ITenantServiceService_IGetTenantSummary;
    assertTenant: ITenantServiceService_IAssertTenant;
    assertBulkTenant: ITenantServiceService_IAssertBulkTenant;
    deleteTenant: ITenantServiceService_IDeleteTenant;
    getTenants: ITenantServiceService_IGetTenants;
}

interface ITenantServiceService_IGetTenantInfo extends grpc.MethodDefinition<tenant_pb.TenantInfoRequest, tenant_pb.TenantInfoResponse> {
    path: "/tenant.TenantService/GetTenantInfo";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.TenantInfoRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.TenantInfoRequest>;
    responseSerialize: grpc.serialize<tenant_pb.TenantInfoResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.TenantInfoResponse>;
}
interface ITenantServiceService_IGetTenantSummary extends grpc.MethodDefinition<tenant_pb.TenantSummaryRequest, tenant_pb.TenantSummaryResponse> {
    path: "/tenant.TenantService/GetTenantSummary";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.TenantSummaryRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.TenantSummaryRequest>;
    responseSerialize: grpc.serialize<tenant_pb.TenantSummaryResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.TenantSummaryResponse>;
}
interface ITenantServiceService_IAssertTenant extends grpc.MethodDefinition<tenant_pb.AssertTenantRequest, tenant_pb.AssertTenantResponse> {
    path: "/tenant.TenantService/AssertTenant";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.AssertTenantRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.AssertTenantRequest>;
    responseSerialize: grpc.serialize<tenant_pb.AssertTenantResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.AssertTenantResponse>;
}
interface ITenantServiceService_IAssertBulkTenant extends grpc.MethodDefinition<tenant_pb.AssertBulkTenantRequest, tenant_pb.AssertBulkTenantResponse> {
    path: "/tenant.TenantService/AssertBulkTenant";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.AssertBulkTenantRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.AssertBulkTenantRequest>;
    responseSerialize: grpc.serialize<tenant_pb.AssertBulkTenantResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.AssertBulkTenantResponse>;
}
interface ITenantServiceService_IDeleteTenant extends grpc.MethodDefinition<tenant_pb.DeleteTenantRequest, tenant_pb.DeleteTenantResponse> {
    path: "/tenant.TenantService/DeleteTenant";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.DeleteTenantRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.DeleteTenantRequest>;
    responseSerialize: grpc.serialize<tenant_pb.DeleteTenantResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.DeleteTenantResponse>;
}
interface ITenantServiceService_IGetTenants extends grpc.MethodDefinition<tenant_pb.GetTenantsRequest, tenant_pb.GetTenantsResponse> {
    path: "/tenant.TenantService/GetTenants";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tenant_pb.GetTenantsRequest>;
    requestDeserialize: grpc.deserialize<tenant_pb.GetTenantsRequest>;
    responseSerialize: grpc.serialize<tenant_pb.GetTenantsResponse>;
    responseDeserialize: grpc.deserialize<tenant_pb.GetTenantsResponse>;
}

export const TenantServiceService: ITenantServiceService;

export interface ITenantServiceServer extends grpc.UntypedServiceImplementation {
    getTenantInfo: grpc.handleUnaryCall<tenant_pb.TenantInfoRequest, tenant_pb.TenantInfoResponse>;
    getTenantSummary: grpc.handleUnaryCall<tenant_pb.TenantSummaryRequest, tenant_pb.TenantSummaryResponse>;
    assertTenant: grpc.handleUnaryCall<tenant_pb.AssertTenantRequest, tenant_pb.AssertTenantResponse>;
    assertBulkTenant: grpc.handleUnaryCall<tenant_pb.AssertBulkTenantRequest, tenant_pb.AssertBulkTenantResponse>;
    deleteTenant: grpc.handleUnaryCall<tenant_pb.DeleteTenantRequest, tenant_pb.DeleteTenantResponse>;
    getTenants: grpc.handleUnaryCall<tenant_pb.GetTenantsRequest, tenant_pb.GetTenantsResponse>;
}

export interface ITenantServiceClient {
    getTenantInfo(request: tenant_pb.TenantInfoRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    getTenantInfo(request: tenant_pb.TenantInfoRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    getTenantInfo(request: tenant_pb.TenantInfoRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    getTenantSummary(request: tenant_pb.TenantSummaryRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    getTenantSummary(request: tenant_pb.TenantSummaryRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    getTenantSummary(request: tenant_pb.TenantSummaryRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    assertTenant(request: tenant_pb.AssertTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    assertTenant(request: tenant_pb.AssertTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    assertTenant(request: tenant_pb.AssertTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    deleteTenant(request: tenant_pb.DeleteTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    deleteTenant(request: tenant_pb.DeleteTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    deleteTenant(request: tenant_pb.DeleteTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    getTenants(request: tenant_pb.GetTenantsRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
    getTenants(request: tenant_pb.GetTenantsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
    getTenants(request: tenant_pb.GetTenantsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
}

export class TenantServiceClient extends grpc.Client implements ITenantServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public getTenantInfo(request: tenant_pb.TenantInfoRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    public getTenantInfo(request: tenant_pb.TenantInfoRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    public getTenantInfo(request: tenant_pb.TenantInfoRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantInfoResponse) => void): grpc.ClientUnaryCall;
    public getTenantSummary(request: tenant_pb.TenantSummaryRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    public getTenantSummary(request: tenant_pb.TenantSummaryRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    public getTenantSummary(request: tenant_pb.TenantSummaryRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.TenantSummaryResponse) => void): grpc.ClientUnaryCall;
    public assertTenant(request: tenant_pb.AssertTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    public assertTenant(request: tenant_pb.AssertTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    public assertTenant(request: tenant_pb.AssertTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertTenantResponse) => void): grpc.ClientUnaryCall;
    public assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    public assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    public assertBulkTenant(request: tenant_pb.AssertBulkTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.AssertBulkTenantResponse) => void): grpc.ClientUnaryCall;
    public deleteTenant(request: tenant_pb.DeleteTenantRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    public deleteTenant(request: tenant_pb.DeleteTenantRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    public deleteTenant(request: tenant_pb.DeleteTenantRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.DeleteTenantResponse) => void): grpc.ClientUnaryCall;
    public getTenants(request: tenant_pb.GetTenantsRequest, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
    public getTenants(request: tenant_pb.GetTenantsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
    public getTenants(request: tenant_pb.GetTenantsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tenant_pb.GetTenantsResponse) => void): grpc.ClientUnaryCall;
}
