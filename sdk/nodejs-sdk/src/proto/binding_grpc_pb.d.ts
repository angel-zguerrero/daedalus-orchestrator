// package: binding
// file: binding.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as binding_pb from "./binding_pb";

interface IBindingServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createBinding: IBindingServiceService_ICreateBinding;
    getBinding: IBindingServiceService_IGetBinding;
    getBindings: IBindingServiceService_IGetBindings;
    deleteBinding: IBindingServiceService_IDeleteBinding;
}

interface IBindingServiceService_ICreateBinding extends grpc.MethodDefinition<binding_pb.CreateBindingRequest, binding_pb.CreateBindingResponse> {
    path: "/binding.BindingService/CreateBinding";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<binding_pb.CreateBindingRequest>;
    requestDeserialize: grpc.deserialize<binding_pb.CreateBindingRequest>;
    responseSerialize: grpc.serialize<binding_pb.CreateBindingResponse>;
    responseDeserialize: grpc.deserialize<binding_pb.CreateBindingResponse>;
}
interface IBindingServiceService_IGetBinding extends grpc.MethodDefinition<binding_pb.GetBindingRequest, binding_pb.GetBindingResponse> {
    path: "/binding.BindingService/GetBinding";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<binding_pb.GetBindingRequest>;
    requestDeserialize: grpc.deserialize<binding_pb.GetBindingRequest>;
    responseSerialize: grpc.serialize<binding_pb.GetBindingResponse>;
    responseDeserialize: grpc.deserialize<binding_pb.GetBindingResponse>;
}
interface IBindingServiceService_IGetBindings extends grpc.MethodDefinition<binding_pb.GetBindingsRequest, binding_pb.GetBindingsResponse> {
    path: "/binding.BindingService/GetBindings";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<binding_pb.GetBindingsRequest>;
    requestDeserialize: grpc.deserialize<binding_pb.GetBindingsRequest>;
    responseSerialize: grpc.serialize<binding_pb.GetBindingsResponse>;
    responseDeserialize: grpc.deserialize<binding_pb.GetBindingsResponse>;
}
interface IBindingServiceService_IDeleteBinding extends grpc.MethodDefinition<binding_pb.DeleteBindingRequest, binding_pb.DeleteBindingResponse> {
    path: "/binding.BindingService/DeleteBinding";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<binding_pb.DeleteBindingRequest>;
    requestDeserialize: grpc.deserialize<binding_pb.DeleteBindingRequest>;
    responseSerialize: grpc.serialize<binding_pb.DeleteBindingResponse>;
    responseDeserialize: grpc.deserialize<binding_pb.DeleteBindingResponse>;
}

export const BindingServiceService: IBindingServiceService;

export interface IBindingServiceServer extends grpc.UntypedServiceImplementation {
    createBinding: grpc.handleUnaryCall<binding_pb.CreateBindingRequest, binding_pb.CreateBindingResponse>;
    getBinding: grpc.handleUnaryCall<binding_pb.GetBindingRequest, binding_pb.GetBindingResponse>;
    getBindings: grpc.handleUnaryCall<binding_pb.GetBindingsRequest, binding_pb.GetBindingsResponse>;
    deleteBinding: grpc.handleUnaryCall<binding_pb.DeleteBindingRequest, binding_pb.DeleteBindingResponse>;
}

export interface IBindingServiceClient {
    createBinding(request: binding_pb.CreateBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    createBinding(request: binding_pb.CreateBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    createBinding(request: binding_pb.CreateBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    getBinding(request: binding_pb.GetBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    getBinding(request: binding_pb.GetBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    getBinding(request: binding_pb.GetBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    getBindings(request: binding_pb.GetBindingsRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    getBindings(request: binding_pb.GetBindingsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    getBindings(request: binding_pb.GetBindingsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    deleteBinding(request: binding_pb.DeleteBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
    deleteBinding(request: binding_pb.DeleteBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
    deleteBinding(request: binding_pb.DeleteBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
}

export class BindingServiceClient extends grpc.Client implements IBindingServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createBinding(request: binding_pb.CreateBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    public createBinding(request: binding_pb.CreateBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    public createBinding(request: binding_pb.CreateBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.CreateBindingResponse) => void): grpc.ClientUnaryCall;
    public getBinding(request: binding_pb.GetBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    public getBinding(request: binding_pb.GetBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    public getBinding(request: binding_pb.GetBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingResponse) => void): grpc.ClientUnaryCall;
    public getBindings(request: binding_pb.GetBindingsRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    public getBindings(request: binding_pb.GetBindingsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    public getBindings(request: binding_pb.GetBindingsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.GetBindingsResponse) => void): grpc.ClientUnaryCall;
    public deleteBinding(request: binding_pb.DeleteBindingRequest, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
    public deleteBinding(request: binding_pb.DeleteBindingRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
    public deleteBinding(request: binding_pb.DeleteBindingRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: binding_pb.DeleteBindingResponse) => void): grpc.ClientUnaryCall;
}
