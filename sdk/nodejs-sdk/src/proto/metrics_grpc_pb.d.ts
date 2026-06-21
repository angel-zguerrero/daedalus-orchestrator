// package: metrics
// file: metrics.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as metrics_pb from "./metrics_pb";

interface IMetricsServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    getSystemMetrics: IMetricsServiceService_IGetSystemMetrics;
}

interface IMetricsServiceService_IGetSystemMetrics extends grpc.MethodDefinition<metrics_pb.SystemMetricsRequest, metrics_pb.SystemMetricsResponse> {
    path: "/metrics.MetricsService/GetSystemMetrics";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<metrics_pb.SystemMetricsRequest>;
    requestDeserialize: grpc.deserialize<metrics_pb.SystemMetricsRequest>;
    responseSerialize: grpc.serialize<metrics_pb.SystemMetricsResponse>;
    responseDeserialize: grpc.deserialize<metrics_pb.SystemMetricsResponse>;
}

export const MetricsServiceService: IMetricsServiceService;

export interface IMetricsServiceServer extends grpc.UntypedServiceImplementation {
    getSystemMetrics: grpc.handleUnaryCall<metrics_pb.SystemMetricsRequest, metrics_pb.SystemMetricsResponse>;
}

export interface IMetricsServiceClient {
    getSystemMetrics(request: metrics_pb.SystemMetricsRequest, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
    getSystemMetrics(request: metrics_pb.SystemMetricsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
    getSystemMetrics(request: metrics_pb.SystemMetricsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
}

export class MetricsServiceClient extends grpc.Client implements IMetricsServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public getSystemMetrics(request: metrics_pb.SystemMetricsRequest, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
    public getSystemMetrics(request: metrics_pb.SystemMetricsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
    public getSystemMetrics(request: metrics_pb.SystemMetricsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: metrics_pb.SystemMetricsResponse) => void): grpc.ClientUnaryCall;
}
