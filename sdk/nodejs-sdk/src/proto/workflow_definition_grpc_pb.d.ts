// package: workflowdefinition
// file: workflow_definition.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as workflow_definition_pb from "./workflow_definition_pb";

interface IWorkflowDefinitionServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createWorkflowDefinition: IWorkflowDefinitionServiceService_ICreateWorkflowDefinition;
    updateWorkflowDefinition: IWorkflowDefinitionServiceService_IUpdateWorkflowDefinition;
    getWorkflowDefinition: IWorkflowDefinitionServiceService_IGetWorkflowDefinition;
    listWorkflowDefinitions: IWorkflowDefinitionServiceService_IListWorkflowDefinitions;
    deleteWorkflowDefinition: IWorkflowDefinitionServiceService_IDeleteWorkflowDefinition;
}

interface IWorkflowDefinitionServiceService_ICreateWorkflowDefinition extends grpc.MethodDefinition<workflow_definition_pb.CreateWorkflowDefinitionRequest, workflow_definition_pb.CreateWorkflowDefinitionResponse> {
    path: "/workflowdefinition.WorkflowDefinitionService/CreateWorkflowDefinition";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflow_definition_pb.CreateWorkflowDefinitionRequest>;
    requestDeserialize: grpc.deserialize<workflow_definition_pb.CreateWorkflowDefinitionRequest>;
    responseSerialize: grpc.serialize<workflow_definition_pb.CreateWorkflowDefinitionResponse>;
    responseDeserialize: grpc.deserialize<workflow_definition_pb.CreateWorkflowDefinitionResponse>;
}
interface IWorkflowDefinitionServiceService_IUpdateWorkflowDefinition extends grpc.MethodDefinition<workflow_definition_pb.UpdateWorkflowDefinitionRequest, workflow_definition_pb.UpdateWorkflowDefinitionResponse> {
    path: "/workflowdefinition.WorkflowDefinitionService/UpdateWorkflowDefinition";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflow_definition_pb.UpdateWorkflowDefinitionRequest>;
    requestDeserialize: grpc.deserialize<workflow_definition_pb.UpdateWorkflowDefinitionRequest>;
    responseSerialize: grpc.serialize<workflow_definition_pb.UpdateWorkflowDefinitionResponse>;
    responseDeserialize: grpc.deserialize<workflow_definition_pb.UpdateWorkflowDefinitionResponse>;
}
interface IWorkflowDefinitionServiceService_IGetWorkflowDefinition extends grpc.MethodDefinition<workflow_definition_pb.GetWorkflowDefinitionRequest, workflow_definition_pb.GetWorkflowDefinitionResponse> {
    path: "/workflowdefinition.WorkflowDefinitionService/GetWorkflowDefinition";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflow_definition_pb.GetWorkflowDefinitionRequest>;
    requestDeserialize: grpc.deserialize<workflow_definition_pb.GetWorkflowDefinitionRequest>;
    responseSerialize: grpc.serialize<workflow_definition_pb.GetWorkflowDefinitionResponse>;
    responseDeserialize: grpc.deserialize<workflow_definition_pb.GetWorkflowDefinitionResponse>;
}
interface IWorkflowDefinitionServiceService_IListWorkflowDefinitions extends grpc.MethodDefinition<workflow_definition_pb.ListWorkflowDefinitionsRequest, workflow_definition_pb.ListWorkflowDefinitionsResponse> {
    path: "/workflowdefinition.WorkflowDefinitionService/ListWorkflowDefinitions";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflow_definition_pb.ListWorkflowDefinitionsRequest>;
    requestDeserialize: grpc.deserialize<workflow_definition_pb.ListWorkflowDefinitionsRequest>;
    responseSerialize: grpc.serialize<workflow_definition_pb.ListWorkflowDefinitionsResponse>;
    responseDeserialize: grpc.deserialize<workflow_definition_pb.ListWorkflowDefinitionsResponse>;
}
interface IWorkflowDefinitionServiceService_IDeleteWorkflowDefinition extends grpc.MethodDefinition<workflow_definition_pb.DeleteWorkflowDefinitionRequest, workflow_definition_pb.DeleteWorkflowDefinitionResponse> {
    path: "/workflowdefinition.WorkflowDefinitionService/DeleteWorkflowDefinition";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflow_definition_pb.DeleteWorkflowDefinitionRequest>;
    requestDeserialize: grpc.deserialize<workflow_definition_pb.DeleteWorkflowDefinitionRequest>;
    responseSerialize: grpc.serialize<workflow_definition_pb.DeleteWorkflowDefinitionResponse>;
    responseDeserialize: grpc.deserialize<workflow_definition_pb.DeleteWorkflowDefinitionResponse>;
}

export const WorkflowDefinitionServiceService: IWorkflowDefinitionServiceService;

export interface IWorkflowDefinitionServiceServer extends grpc.UntypedServiceImplementation {
    createWorkflowDefinition: grpc.handleUnaryCall<workflow_definition_pb.CreateWorkflowDefinitionRequest, workflow_definition_pb.CreateWorkflowDefinitionResponse>;
    updateWorkflowDefinition: grpc.handleUnaryCall<workflow_definition_pb.UpdateWorkflowDefinitionRequest, workflow_definition_pb.UpdateWorkflowDefinitionResponse>;
    getWorkflowDefinition: grpc.handleUnaryCall<workflow_definition_pb.GetWorkflowDefinitionRequest, workflow_definition_pb.GetWorkflowDefinitionResponse>;
    listWorkflowDefinitions: grpc.handleUnaryCall<workflow_definition_pb.ListWorkflowDefinitionsRequest, workflow_definition_pb.ListWorkflowDefinitionsResponse>;
    deleteWorkflowDefinition: grpc.handleUnaryCall<workflow_definition_pb.DeleteWorkflowDefinitionRequest, workflow_definition_pb.DeleteWorkflowDefinitionResponse>;
}

export interface IWorkflowDefinitionServiceClient {
    createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
}

export class WorkflowDefinitionServiceClient extends grpc.Client implements IWorkflowDefinitionServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public createWorkflowDefinition(request: workflow_definition_pb.CreateWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.CreateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public updateWorkflowDefinition(request: workflow_definition_pb.UpdateWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.UpdateWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public getWorkflowDefinition(request: workflow_definition_pb.GetWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.GetWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflowDefinitions(request: workflow_definition_pb.ListWorkflowDefinitionsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.ListWorkflowDefinitionsResponse) => void): grpc.ClientUnaryCall;
    public deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
    public deleteWorkflowDefinition(request: workflow_definition_pb.DeleteWorkflowDefinitionRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflow_definition_pb.DeleteWorkflowDefinitionResponse) => void): grpc.ClientUnaryCall;
}
