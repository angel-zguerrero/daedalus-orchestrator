// package: jobworker
// file: jobworker.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as jobworker_pb from "./jobworker_pb";

interface IJobWorkerServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    claimWork: IJobWorkerServiceService_IClaimWork;
    ackMessage: IJobWorkerServiceService_IAckMessage;
    bulkAckMessages: IJobWorkerServiceService_IBulkAckMessages;
}

interface IJobWorkerServiceService_IClaimWork extends grpc.MethodDefinition<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage> {
    path: "/jobworker.JobWorkerService/ClaimWork";
    requestStream: true;
    responseStream: true;
    requestSerialize: grpc.serialize<jobworker_pb.ClaimWorkRequest>;
    requestDeserialize: grpc.deserialize<jobworker_pb.ClaimWorkRequest>;
    responseSerialize: grpc.serialize<jobworker_pb.ClaimWorkStreamMessage>;
    responseDeserialize: grpc.deserialize<jobworker_pb.ClaimWorkStreamMessage>;
}
interface IJobWorkerServiceService_IAckMessage extends grpc.MethodDefinition<jobworker_pb.AckMessageRequest, jobworker_pb.AckMessageResponse> {
    path: "/jobworker.JobWorkerService/AckMessage";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<jobworker_pb.AckMessageRequest>;
    requestDeserialize: grpc.deserialize<jobworker_pb.AckMessageRequest>;
    responseSerialize: grpc.serialize<jobworker_pb.AckMessageResponse>;
    responseDeserialize: grpc.deserialize<jobworker_pb.AckMessageResponse>;
}
interface IJobWorkerServiceService_IBulkAckMessages extends grpc.MethodDefinition<jobworker_pb.BulkAckMessageRequest, jobworker_pb.AckMessageResponse> {
    path: "/jobworker.JobWorkerService/BulkAckMessages";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<jobworker_pb.BulkAckMessageRequest>;
    requestDeserialize: grpc.deserialize<jobworker_pb.BulkAckMessageRequest>;
    responseSerialize: grpc.serialize<jobworker_pb.AckMessageResponse>;
    responseDeserialize: grpc.deserialize<jobworker_pb.AckMessageResponse>;
}

export const JobWorkerServiceService: IJobWorkerServiceService;

export interface IJobWorkerServiceServer extends grpc.UntypedServiceImplementation {
    claimWork: grpc.handleBidiStreamingCall<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    ackMessage: grpc.handleUnaryCall<jobworker_pb.AckMessageRequest, jobworker_pb.AckMessageResponse>;
    bulkAckMessages: grpc.handleUnaryCall<jobworker_pb.BulkAckMessageRequest, jobworker_pb.AckMessageResponse>;
}

export interface IJobWorkerServiceClient {
    claimWork(): grpc.ClientDuplexStream<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    claimWork(options: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    claimWork(metadata: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    ackMessage(request: jobworker_pb.AckMessageRequest, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    ackMessage(request: jobworker_pb.AckMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    ackMessage(request: jobworker_pb.AckMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
}

export class JobWorkerServiceClient extends grpc.Client implements IJobWorkerServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public claimWork(options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    public claimWork(metadata?: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<jobworker_pb.ClaimWorkRequest, jobworker_pb.ClaimWorkStreamMessage>;
    public ackMessage(request: jobworker_pb.AckMessageRequest, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    public ackMessage(request: jobworker_pb.AckMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    public ackMessage(request: jobworker_pb.AckMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    public bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    public bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
    public bulkAckMessages(request: jobworker_pb.BulkAckMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: jobworker_pb.AckMessageResponse) => void): grpc.ClientUnaryCall;
}
