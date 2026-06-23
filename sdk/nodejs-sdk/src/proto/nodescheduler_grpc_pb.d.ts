// package: nodescheduler
// file: nodescheduler.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as nodescheduler_pb from "./nodescheduler_pb";

interface INodeSchedulerServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    getNodeSchedulers: INodeSchedulerServiceService_IGetNodeSchedulers;
    getNodeScheduler: INodeSchedulerServiceService_IGetNodeScheduler;
}

interface INodeSchedulerServiceService_IGetNodeSchedulers extends grpc.MethodDefinition<nodescheduler_pb.GetNodeSchedulersRequest, nodescheduler_pb.GetNodeSchedulersResponse> {
    path: "/nodescheduler.NodeSchedulerService/GetNodeSchedulers";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<nodescheduler_pb.GetNodeSchedulersRequest>;
    requestDeserialize: grpc.deserialize<nodescheduler_pb.GetNodeSchedulersRequest>;
    responseSerialize: grpc.serialize<nodescheduler_pb.GetNodeSchedulersResponse>;
    responseDeserialize: grpc.deserialize<nodescheduler_pb.GetNodeSchedulersResponse>;
}
interface INodeSchedulerServiceService_IGetNodeScheduler extends grpc.MethodDefinition<nodescheduler_pb.GetNodeSchedulerRequest, nodescheduler_pb.GetNodeSchedulerResponse> {
    path: "/nodescheduler.NodeSchedulerService/GetNodeScheduler";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<nodescheduler_pb.GetNodeSchedulerRequest>;
    requestDeserialize: grpc.deserialize<nodescheduler_pb.GetNodeSchedulerRequest>;
    responseSerialize: grpc.serialize<nodescheduler_pb.GetNodeSchedulerResponse>;
    responseDeserialize: grpc.deserialize<nodescheduler_pb.GetNodeSchedulerResponse>;
}

export const NodeSchedulerServiceService: INodeSchedulerServiceService;

export interface INodeSchedulerServiceServer extends grpc.UntypedServiceImplementation {
    getNodeSchedulers: grpc.handleUnaryCall<nodescheduler_pb.GetNodeSchedulersRequest, nodescheduler_pb.GetNodeSchedulersResponse>;
    getNodeScheduler: grpc.handleUnaryCall<nodescheduler_pb.GetNodeSchedulerRequest, nodescheduler_pb.GetNodeSchedulerResponse>;
}

export interface INodeSchedulerServiceClient {
    getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
    getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
    getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
}

export class NodeSchedulerServiceClient extends grpc.Client implements INodeSchedulerServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    public getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    public getNodeSchedulers(request: nodescheduler_pb.GetNodeSchedulersRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulersResponse) => void): grpc.ClientUnaryCall;
    public getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
    public getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
    public getNodeScheduler(request: nodescheduler_pb.GetNodeSchedulerRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: nodescheduler_pb.GetNodeSchedulerResponse) => void): grpc.ClientUnaryCall;
}
