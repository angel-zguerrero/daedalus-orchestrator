// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var binding_pb = require('./binding_pb.js');

function serialize_binding_BulkCreateBindingRequest(arg) {
  if (!(arg instanceof binding_pb.BulkCreateBindingRequest)) {
    throw new Error('Expected argument of type binding.BulkCreateBindingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_BulkCreateBindingRequest(buffer_arg) {
  return binding_pb.BulkCreateBindingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_BulkCreateBindingResponse(arg) {
  if (!(arg instanceof binding_pb.BulkCreateBindingResponse)) {
    throw new Error('Expected argument of type binding.BulkCreateBindingResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_BulkCreateBindingResponse(buffer_arg) {
  return binding_pb.BulkCreateBindingResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_CreateBindingRequest(arg) {
  if (!(arg instanceof binding_pb.CreateBindingRequest)) {
    throw new Error('Expected argument of type binding.CreateBindingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_CreateBindingRequest(buffer_arg) {
  return binding_pb.CreateBindingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_CreateBindingResponse(arg) {
  if (!(arg instanceof binding_pb.CreateBindingResponse)) {
    throw new Error('Expected argument of type binding.CreateBindingResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_CreateBindingResponse(buffer_arg) {
  return binding_pb.CreateBindingResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_DeleteBindingRequest(arg) {
  if (!(arg instanceof binding_pb.DeleteBindingRequest)) {
    throw new Error('Expected argument of type binding.DeleteBindingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_DeleteBindingRequest(buffer_arg) {
  return binding_pb.DeleteBindingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_DeleteBindingResponse(arg) {
  if (!(arg instanceof binding_pb.DeleteBindingResponse)) {
    throw new Error('Expected argument of type binding.DeleteBindingResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_DeleteBindingResponse(buffer_arg) {
  return binding_pb.DeleteBindingResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_GetBindingRequest(arg) {
  if (!(arg instanceof binding_pb.GetBindingRequest)) {
    throw new Error('Expected argument of type binding.GetBindingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_GetBindingRequest(buffer_arg) {
  return binding_pb.GetBindingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_GetBindingResponse(arg) {
  if (!(arg instanceof binding_pb.GetBindingResponse)) {
    throw new Error('Expected argument of type binding.GetBindingResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_GetBindingResponse(buffer_arg) {
  return binding_pb.GetBindingResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_GetBindingsRequest(arg) {
  if (!(arg instanceof binding_pb.GetBindingsRequest)) {
    throw new Error('Expected argument of type binding.GetBindingsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_GetBindingsRequest(buffer_arg) {
  return binding_pb.GetBindingsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_binding_GetBindingsResponse(arg) {
  if (!(arg instanceof binding_pb.GetBindingsResponse)) {
    throw new Error('Expected argument of type binding.GetBindingsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_binding_GetBindingsResponse(buffer_arg) {
  return binding_pb.GetBindingsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var BindingServiceService = exports.BindingServiceService = {
  createBinding: {
    path: '/binding.BindingService/CreateBinding',
    requestStream: false,
    responseStream: false,
    requestType: binding_pb.CreateBindingRequest,
    responseType: binding_pb.CreateBindingResponse,
    requestSerialize: serialize_binding_CreateBindingRequest,
    requestDeserialize: deserialize_binding_CreateBindingRequest,
    responseSerialize: serialize_binding_CreateBindingResponse,
    responseDeserialize: deserialize_binding_CreateBindingResponse,
  },
  bulkCreateBinding: {
    path: '/binding.BindingService/BulkCreateBinding',
    requestStream: false,
    responseStream: false,
    requestType: binding_pb.BulkCreateBindingRequest,
    responseType: binding_pb.BulkCreateBindingResponse,
    requestSerialize: serialize_binding_BulkCreateBindingRequest,
    requestDeserialize: deserialize_binding_BulkCreateBindingRequest,
    responseSerialize: serialize_binding_BulkCreateBindingResponse,
    responseDeserialize: deserialize_binding_BulkCreateBindingResponse,
  },
  getBinding: {
    path: '/binding.BindingService/GetBinding',
    requestStream: false,
    responseStream: false,
    requestType: binding_pb.GetBindingRequest,
    responseType: binding_pb.GetBindingResponse,
    requestSerialize: serialize_binding_GetBindingRequest,
    requestDeserialize: deserialize_binding_GetBindingRequest,
    responseSerialize: serialize_binding_GetBindingResponse,
    responseDeserialize: deserialize_binding_GetBindingResponse,
  },
  getBindings: {
    path: '/binding.BindingService/GetBindings',
    requestStream: false,
    responseStream: false,
    requestType: binding_pb.GetBindingsRequest,
    responseType: binding_pb.GetBindingsResponse,
    requestSerialize: serialize_binding_GetBindingsRequest,
    requestDeserialize: deserialize_binding_GetBindingsRequest,
    responseSerialize: serialize_binding_GetBindingsResponse,
    responseDeserialize: deserialize_binding_GetBindingsResponse,
  },
  deleteBinding: {
    path: '/binding.BindingService/DeleteBinding',
    requestStream: false,
    responseStream: false,
    requestType: binding_pb.DeleteBindingRequest,
    responseType: binding_pb.DeleteBindingResponse,
    requestSerialize: serialize_binding_DeleteBindingRequest,
    requestDeserialize: deserialize_binding_DeleteBindingRequest,
    responseSerialize: serialize_binding_DeleteBindingResponse,
    responseDeserialize: deserialize_binding_DeleteBindingResponse,
  },
};

exports.BindingServiceClient = grpc.makeGenericClientConstructor(BindingServiceService, 'BindingService');
