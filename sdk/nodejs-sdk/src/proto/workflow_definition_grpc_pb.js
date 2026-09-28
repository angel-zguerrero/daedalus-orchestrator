// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var workflow_definition_pb = require('./workflow_definition_pb.js');

function serialize_workflowdefinition_CreateWorkflowDefinitionRequest(arg) {
  if (!(arg instanceof workflow_definition_pb.CreateWorkflowDefinitionRequest)) {
    throw new Error('Expected argument of type workflowdefinition.CreateWorkflowDefinitionRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_CreateWorkflowDefinitionRequest(buffer_arg) {
  return workflow_definition_pb.CreateWorkflowDefinitionRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_CreateWorkflowDefinitionResponse(arg) {
  if (!(arg instanceof workflow_definition_pb.CreateWorkflowDefinitionResponse)) {
    throw new Error('Expected argument of type workflowdefinition.CreateWorkflowDefinitionResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_CreateWorkflowDefinitionResponse(buffer_arg) {
  return workflow_definition_pb.CreateWorkflowDefinitionResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_DeleteWorkflowDefinitionRequest(arg) {
  if (!(arg instanceof workflow_definition_pb.DeleteWorkflowDefinitionRequest)) {
    throw new Error('Expected argument of type workflowdefinition.DeleteWorkflowDefinitionRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_DeleteWorkflowDefinitionRequest(buffer_arg) {
  return workflow_definition_pb.DeleteWorkflowDefinitionRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_DeleteWorkflowDefinitionResponse(arg) {
  if (!(arg instanceof workflow_definition_pb.DeleteWorkflowDefinitionResponse)) {
    throw new Error('Expected argument of type workflowdefinition.DeleteWorkflowDefinitionResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_DeleteWorkflowDefinitionResponse(buffer_arg) {
  return workflow_definition_pb.DeleteWorkflowDefinitionResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_GetWorkflowDefinitionRequest(arg) {
  if (!(arg instanceof workflow_definition_pb.GetWorkflowDefinitionRequest)) {
    throw new Error('Expected argument of type workflowdefinition.GetWorkflowDefinitionRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_GetWorkflowDefinitionRequest(buffer_arg) {
  return workflow_definition_pb.GetWorkflowDefinitionRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_GetWorkflowDefinitionResponse(arg) {
  if (!(arg instanceof workflow_definition_pb.GetWorkflowDefinitionResponse)) {
    throw new Error('Expected argument of type workflowdefinition.GetWorkflowDefinitionResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_GetWorkflowDefinitionResponse(buffer_arg) {
  return workflow_definition_pb.GetWorkflowDefinitionResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_ListWorkflowDefinitionsRequest(arg) {
  if (!(arg instanceof workflow_definition_pb.ListWorkflowDefinitionsRequest)) {
    throw new Error('Expected argument of type workflowdefinition.ListWorkflowDefinitionsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_ListWorkflowDefinitionsRequest(buffer_arg) {
  return workflow_definition_pb.ListWorkflowDefinitionsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_ListWorkflowDefinitionsResponse(arg) {
  if (!(arg instanceof workflow_definition_pb.ListWorkflowDefinitionsResponse)) {
    throw new Error('Expected argument of type workflowdefinition.ListWorkflowDefinitionsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_ListWorkflowDefinitionsResponse(buffer_arg) {
  return workflow_definition_pb.ListWorkflowDefinitionsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_UpdateWorkflowDefinitionRequest(arg) {
  if (!(arg instanceof workflow_definition_pb.UpdateWorkflowDefinitionRequest)) {
    throw new Error('Expected argument of type workflowdefinition.UpdateWorkflowDefinitionRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_UpdateWorkflowDefinitionRequest(buffer_arg) {
  return workflow_definition_pb.UpdateWorkflowDefinitionRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_workflowdefinition_UpdateWorkflowDefinitionResponse(arg) {
  if (!(arg instanceof workflow_definition_pb.UpdateWorkflowDefinitionResponse)) {
    throw new Error('Expected argument of type workflowdefinition.UpdateWorkflowDefinitionResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_workflowdefinition_UpdateWorkflowDefinitionResponse(buffer_arg) {
  return workflow_definition_pb.UpdateWorkflowDefinitionResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var WorkflowDefinitionServiceService = exports.WorkflowDefinitionServiceService = {
  createWorkflowDefinition: {
    path: '/workflowdefinition.WorkflowDefinitionService/CreateWorkflowDefinition',
    requestStream: false,
    responseStream: false,
    requestType: workflow_definition_pb.CreateWorkflowDefinitionRequest,
    responseType: workflow_definition_pb.CreateWorkflowDefinitionResponse,
    requestSerialize: serialize_workflowdefinition_CreateWorkflowDefinitionRequest,
    requestDeserialize: deserialize_workflowdefinition_CreateWorkflowDefinitionRequest,
    responseSerialize: serialize_workflowdefinition_CreateWorkflowDefinitionResponse,
    responseDeserialize: deserialize_workflowdefinition_CreateWorkflowDefinitionResponse,
  },
  updateWorkflowDefinition: {
    path: '/workflowdefinition.WorkflowDefinitionService/UpdateWorkflowDefinition',
    requestStream: false,
    responseStream: false,
    requestType: workflow_definition_pb.UpdateWorkflowDefinitionRequest,
    responseType: workflow_definition_pb.UpdateWorkflowDefinitionResponse,
    requestSerialize: serialize_workflowdefinition_UpdateWorkflowDefinitionRequest,
    requestDeserialize: deserialize_workflowdefinition_UpdateWorkflowDefinitionRequest,
    responseSerialize: serialize_workflowdefinition_UpdateWorkflowDefinitionResponse,
    responseDeserialize: deserialize_workflowdefinition_UpdateWorkflowDefinitionResponse,
  },
  getWorkflowDefinition: {
    path: '/workflowdefinition.WorkflowDefinitionService/GetWorkflowDefinition',
    requestStream: false,
    responseStream: false,
    requestType: workflow_definition_pb.GetWorkflowDefinitionRequest,
    responseType: workflow_definition_pb.GetWorkflowDefinitionResponse,
    requestSerialize: serialize_workflowdefinition_GetWorkflowDefinitionRequest,
    requestDeserialize: deserialize_workflowdefinition_GetWorkflowDefinitionRequest,
    responseSerialize: serialize_workflowdefinition_GetWorkflowDefinitionResponse,
    responseDeserialize: deserialize_workflowdefinition_GetWorkflowDefinitionResponse,
  },
  listWorkflowDefinitions: {
    path: '/workflowdefinition.WorkflowDefinitionService/ListWorkflowDefinitions',
    requestStream: false,
    responseStream: false,
    requestType: workflow_definition_pb.ListWorkflowDefinitionsRequest,
    responseType: workflow_definition_pb.ListWorkflowDefinitionsResponse,
    requestSerialize: serialize_workflowdefinition_ListWorkflowDefinitionsRequest,
    requestDeserialize: deserialize_workflowdefinition_ListWorkflowDefinitionsRequest,
    responseSerialize: serialize_workflowdefinition_ListWorkflowDefinitionsResponse,
    responseDeserialize: deserialize_workflowdefinition_ListWorkflowDefinitionsResponse,
  },
  deleteWorkflowDefinition: {
    path: '/workflowdefinition.WorkflowDefinitionService/DeleteWorkflowDefinition',
    requestStream: false,
    responseStream: false,
    requestType: workflow_definition_pb.DeleteWorkflowDefinitionRequest,
    responseType: workflow_definition_pb.DeleteWorkflowDefinitionResponse,
    requestSerialize: serialize_workflowdefinition_DeleteWorkflowDefinitionRequest,
    requestDeserialize: deserialize_workflowdefinition_DeleteWorkflowDefinitionRequest,
    responseSerialize: serialize_workflowdefinition_DeleteWorkflowDefinitionResponse,
    responseDeserialize: deserialize_workflowdefinition_DeleteWorkflowDefinitionResponse,
  },
};

exports.WorkflowDefinitionServiceClient = grpc.makeGenericClientConstructor(WorkflowDefinitionServiceService, 'WorkflowDefinitionService');
