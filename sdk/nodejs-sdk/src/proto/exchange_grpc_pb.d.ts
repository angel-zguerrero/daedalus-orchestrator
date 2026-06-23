// package: exchange
// file: exchange.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as exchange_pb from "./exchange_pb";

interface IExchangeServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createExchange: IExchangeServiceService_ICreateExchange;
    bulkCreateExchange: IExchangeServiceService_IBulkCreateExchange;
    getExchange: IExchangeServiceService_IGetExchange;
    getExchanges: IExchangeServiceService_IGetExchanges;
    deleteExchange: IExchangeServiceService_IDeleteExchange;
    publishMessage: IExchangeServiceService_IPublishMessage;
    publishStream: IExchangeServiceService_IPublishStream;
}

interface IExchangeServiceService_ICreateExchange extends grpc.MethodDefinition<exchange_pb.CreateExchangeRequest, exchange_pb.CreateExchangeResponse> {
    path: "/exchange.ExchangeService/CreateExchange";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.CreateExchangeRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.CreateExchangeRequest>;
    responseSerialize: grpc.serialize<exchange_pb.CreateExchangeResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.CreateExchangeResponse>;
}
interface IExchangeServiceService_IBulkCreateExchange extends grpc.MethodDefinition<exchange_pb.BulkCreateExchangeRequest, exchange_pb.BulkCreateExchangeResponse> {
    path: "/exchange.ExchangeService/BulkCreateExchange";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.BulkCreateExchangeRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.BulkCreateExchangeRequest>;
    responseSerialize: grpc.serialize<exchange_pb.BulkCreateExchangeResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.BulkCreateExchangeResponse>;
}
interface IExchangeServiceService_IGetExchange extends grpc.MethodDefinition<exchange_pb.GetExchangeRequest, exchange_pb.GetExchangeResponse> {
    path: "/exchange.ExchangeService/GetExchange";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.GetExchangeRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.GetExchangeRequest>;
    responseSerialize: grpc.serialize<exchange_pb.GetExchangeResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.GetExchangeResponse>;
}
interface IExchangeServiceService_IGetExchanges extends grpc.MethodDefinition<exchange_pb.GetExchangesRequest, exchange_pb.GetExchangesResponse> {
    path: "/exchange.ExchangeService/GetExchanges";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.GetExchangesRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.GetExchangesRequest>;
    responseSerialize: grpc.serialize<exchange_pb.GetExchangesResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.GetExchangesResponse>;
}
interface IExchangeServiceService_IDeleteExchange extends grpc.MethodDefinition<exchange_pb.DeleteExchangeRequest, exchange_pb.DeleteExchangeResponse> {
    path: "/exchange.ExchangeService/DeleteExchange";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.DeleteExchangeRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.DeleteExchangeRequest>;
    responseSerialize: grpc.serialize<exchange_pb.DeleteExchangeResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.DeleteExchangeResponse>;
}
interface IExchangeServiceService_IPublishMessage extends grpc.MethodDefinition<exchange_pb.PublishMessageRequest, exchange_pb.PublishMessageResponse> {
    path: "/exchange.ExchangeService/PublishMessage";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<exchange_pb.PublishMessageRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.PublishMessageRequest>;
    responseSerialize: grpc.serialize<exchange_pb.PublishMessageResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.PublishMessageResponse>;
}
interface IExchangeServiceService_IPublishStream extends grpc.MethodDefinition<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse> {
    path: "/exchange.ExchangeService/PublishStream";
    requestStream: true;
    responseStream: true;
    requestSerialize: grpc.serialize<exchange_pb.PublishStreamRequest>;
    requestDeserialize: grpc.deserialize<exchange_pb.PublishStreamRequest>;
    responseSerialize: grpc.serialize<exchange_pb.PublishStreamResponse>;
    responseDeserialize: grpc.deserialize<exchange_pb.PublishStreamResponse>;
}

export const ExchangeServiceService: IExchangeServiceService;

export interface IExchangeServiceServer extends grpc.UntypedServiceImplementation {
    createExchange: grpc.handleUnaryCall<exchange_pb.CreateExchangeRequest, exchange_pb.CreateExchangeResponse>;
    bulkCreateExchange: grpc.handleUnaryCall<exchange_pb.BulkCreateExchangeRequest, exchange_pb.BulkCreateExchangeResponse>;
    getExchange: grpc.handleUnaryCall<exchange_pb.GetExchangeRequest, exchange_pb.GetExchangeResponse>;
    getExchanges: grpc.handleUnaryCall<exchange_pb.GetExchangesRequest, exchange_pb.GetExchangesResponse>;
    deleteExchange: grpc.handleUnaryCall<exchange_pb.DeleteExchangeRequest, exchange_pb.DeleteExchangeResponse>;
    publishMessage: grpc.handleUnaryCall<exchange_pb.PublishMessageRequest, exchange_pb.PublishMessageResponse>;
    publishStream: grpc.handleBidiStreamingCall<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
}

export interface IExchangeServiceClient {
    createExchange(request: exchange_pb.CreateExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    createExchange(request: exchange_pb.CreateExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    createExchange(request: exchange_pb.CreateExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    getExchange(request: exchange_pb.GetExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    getExchange(request: exchange_pb.GetExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    getExchange(request: exchange_pb.GetExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    getExchanges(request: exchange_pb.GetExchangesRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    getExchanges(request: exchange_pb.GetExchangesRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    getExchanges(request: exchange_pb.GetExchangesRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    deleteExchange(request: exchange_pb.DeleteExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    deleteExchange(request: exchange_pb.DeleteExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    deleteExchange(request: exchange_pb.DeleteExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    publishMessage(request: exchange_pb.PublishMessageRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    publishMessage(request: exchange_pb.PublishMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    publishMessage(request: exchange_pb.PublishMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    publishStream(): grpc.ClientDuplexStream<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
    publishStream(options: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
    publishStream(metadata: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
}

export class ExchangeServiceClient extends grpc.Client implements IExchangeServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createExchange(request: exchange_pb.CreateExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public createExchange(request: exchange_pb.CreateExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public createExchange(request: exchange_pb.CreateExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.CreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public bulkCreateExchange(request: exchange_pb.BulkCreateExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.BulkCreateExchangeResponse) => void): grpc.ClientUnaryCall;
    public getExchange(request: exchange_pb.GetExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    public getExchange(request: exchange_pb.GetExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    public getExchange(request: exchange_pb.GetExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangeResponse) => void): grpc.ClientUnaryCall;
    public getExchanges(request: exchange_pb.GetExchangesRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    public getExchanges(request: exchange_pb.GetExchangesRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    public getExchanges(request: exchange_pb.GetExchangesRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.GetExchangesResponse) => void): grpc.ClientUnaryCall;
    public deleteExchange(request: exchange_pb.DeleteExchangeRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    public deleteExchange(request: exchange_pb.DeleteExchangeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    public deleteExchange(request: exchange_pb.DeleteExchangeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.DeleteExchangeResponse) => void): grpc.ClientUnaryCall;
    public publishMessage(request: exchange_pb.PublishMessageRequest, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    public publishMessage(request: exchange_pb.PublishMessageRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    public publishMessage(request: exchange_pb.PublishMessageRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: exchange_pb.PublishMessageResponse) => void): grpc.ClientUnaryCall;
    public publishStream(options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
    public publishStream(metadata?: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientDuplexStream<exchange_pb.PublishStreamRequest, exchange_pb.PublishStreamResponse>;
}
