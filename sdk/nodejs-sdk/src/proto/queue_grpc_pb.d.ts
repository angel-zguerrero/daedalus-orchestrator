// package: queue
// file: queue.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as queue_pb from "./queue_pb";

interface IQueueServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createQueue: IQueueServiceService_ICreateQueue;
    bulkCreateQueue: IQueueServiceService_IBulkCreateQueue;
    getQueue: IQueueServiceService_IGetQueue;
    getQueues: IQueueServiceService_IGetQueues;
    deleteQueue: IQueueServiceService_IDeleteQueue;
    enqueueMessage: IQueueServiceService_IEnqueueMessage;
}

interface IQueueServiceService_ICreateQueue extends grpc.MethodDefinition<queue_pb.CreateQueueRequest, queue_pb.CreateQueueResponse> {
    path: "/queue.QueueService/CreateQueue";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.CreateQueueRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.CreateQueueRequest>;
    responseSerialize: grpc.serialize<queue_pb.CreateQueueResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.CreateQueueResponse>;
}
interface IQueueServiceService_IBulkCreateQueue extends grpc.MethodDefinition<queue_pb.BulkCreateQueueRequest, queue_pb.BulkCreateQueueResponse> {
    path: "/queue.QueueService/BulkCreateQueue";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.BulkCreateQueueRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.BulkCreateQueueRequest>;
    responseSerialize: grpc.serialize<queue_pb.BulkCreateQueueResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.BulkCreateQueueResponse>;
}
interface IQueueServiceService_IGetQueue extends grpc.MethodDefinition<queue_pb.GetQueueRequest, queue_pb.GetQueueResponse> {
    path: "/queue.QueueService/GetQueue";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.GetQueueRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.GetQueueRequest>;
    responseSerialize: grpc.serialize<queue_pb.GetQueueResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.GetQueueResponse>;
}
interface IQueueServiceService_IGetQueues extends grpc.MethodDefinition<queue_pb.GetQueuesRequest, queue_pb.GetQueuesResponse> {
    path: "/queue.QueueService/GetQueues";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.GetQueuesRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.GetQueuesRequest>;
    responseSerialize: grpc.serialize<queue_pb.GetQueuesResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.GetQueuesResponse>;
}
interface IQueueServiceService_IDeleteQueue extends grpc.MethodDefinition<queue_pb.DeleteQueueRequest, queue_pb.DeleteQueueResponse> {
    path: "/queue.QueueService/DeleteQueue";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.DeleteQueueRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.DeleteQueueRequest>;
    responseSerialize: grpc.serialize<queue_pb.DeleteQueueResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.DeleteQueueResponse>;
}
interface IQueueServiceService_IEnqueueMessage extends grpc.MethodDefinition<queue_pb.EnqueueMessageRequest, queue_pb.EnqueueMessageResponse> {
    path: "/queue.QueueService/EnqueueMessage";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<queue_pb.EnqueueMessageRequest>;
    requestDeserialize: grpc.deserialize<queue_pb.EnqueueMessageRequest>;
    responseSerialize: grpc.serialize<queue_pb.EnqueueMessageResponse>;
    responseDeserialize: grpc.deserialize<queue_pb.EnqueueMessageResponse>;
}

export const QueueServiceService: IQueueServiceService;

export interface IQueueServiceServer extends grpc.UntypedServiceImplementation {
    createQueue: grpc.handleUnaryCall<queue_pb.CreateQueueRequest, queue_pb.CreateQueueResponse>;
    bulkCreateQueue: grpc.handleUnaryCall<queue_pb.BulkCreateQueueRequest, queue_pb.BulkCreateQueueResponse>;
    getQueue: grpc.handleUnaryCall<queue_pb.GetQueueRequest, queue_pb.GetQueueResponse>;
    getQueues: grpc.handleUnaryCall<queue_pb.GetQueuesRequest, queue_pb.GetQueuesResponse>;
    deleteQueue: grpc.handleUnaryCall<queue_pb.DeleteQueueRequest, queue_pb.DeleteQueueResponse>;
    enqueueMessage: grpc.handleUnaryCall<queue_pb.EnqueueMessageRequest, queue_pb.EnqueueMessageResponse>;
}

export interface IQueueServiceClient {
    createQueue(request: queue_pb.CreateQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    createQueue(request: queue_pb.CreateQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    createQueue(request: queue_pb.CreateQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    getQueue(request: queue_pb.GetQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    getQueue(request: queue_pb.GetQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    getQueue(request: queue_pb.GetQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    getQueues(request: queue_pb.GetQueuesRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    getQueues(request: queue_pb.GetQueuesRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    getQueues(request: queue_pb.GetQueuesRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    deleteQueue(request: queue_pb.DeleteQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    deleteQueue(request: queue_pb.DeleteQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    deleteQueue(request: queue_pb.DeleteQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    enqueueMessage(request: queue_pb.EnqueueMessageRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
    enqueueMessage(request: queue_pb.EnqueueMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
    enqueueMessage(request: queue_pb.EnqueueMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
}

export class QueueServiceClient extends grpc.Client implements IQueueServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createQueue(request: queue_pb.CreateQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    public createQueue(request: queue_pb.CreateQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    public createQueue(request: queue_pb.CreateQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.CreateQueueResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateQueue(request: queue_pb.BulkCreateQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.BulkCreateQueueResponse) => void): grpc.ClientUnaryCall;
    public getQueue(request: queue_pb.GetQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    public getQueue(request: queue_pb.GetQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    public getQueue(request: queue_pb.GetQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueueResponse) => void): grpc.ClientUnaryCall;
    public getQueues(request: queue_pb.GetQueuesRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    public getQueues(request: queue_pb.GetQueuesRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    public getQueues(request: queue_pb.GetQueuesRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.GetQueuesResponse) => void): grpc.ClientUnaryCall;
    public deleteQueue(request: queue_pb.DeleteQueueRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    public deleteQueue(request: queue_pb.DeleteQueueRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    public deleteQueue(request: queue_pb.DeleteQueueRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.DeleteQueueResponse) => void): grpc.ClientUnaryCall;
    public enqueueMessage(request: queue_pb.EnqueueMessageRequest, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
    public enqueueMessage(request: queue_pb.EnqueueMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
    public enqueueMessage(request: queue_pb.EnqueueMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: queue_pb.EnqueueMessageResponse) => void): grpc.ClientUnaryCall;
}
