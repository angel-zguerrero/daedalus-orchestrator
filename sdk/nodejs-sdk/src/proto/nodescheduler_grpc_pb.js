// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var nodescheduler_pb = require('./nodescheduler_pb.js');

function serialize_nodescheduler_GetNodeSchedulerRequest(arg) {
  if (!(arg instanceof nodescheduler_pb.GetNodeSchedulerRequest)) {
    throw new Error('Expected argument of type nodescheduler.GetNodeSchedulerRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_nodescheduler_GetNodeSchedulerRequest(buffer_arg) {
  return nodescheduler_pb.GetNodeSchedulerRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_nodescheduler_GetNodeSchedulerResponse(arg) {
  if (!(arg instanceof nodescheduler_pb.GetNodeSchedulerResponse)) {
    throw new Error('Expected argument of type nodescheduler.GetNodeSchedulerResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_nodescheduler_GetNodeSchedulerResponse(buffer_arg) {
  return nodescheduler_pb.GetNodeSchedulerResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_nodescheduler_GetNodeSchedulersRequest(arg) {
  if (!(arg instanceof nodescheduler_pb.GetNodeSchedulersRequest)) {
    throw new Error('Expected argument of type nodescheduler.GetNodeSchedulersRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_nodescheduler_GetNodeSchedulersRequest(buffer_arg) {
  return nodescheduler_pb.GetNodeSchedulersRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_nodescheduler_GetNodeSchedulersResponse(arg) {
  if (!(arg instanceof nodescheduler_pb.GetNodeSchedulersResponse)) {
    throw new Error('Expected argument of type nodescheduler.GetNodeSchedulersResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_nodescheduler_GetNodeSchedulersResponse(buffer_arg) {
  return nodescheduler_pb.GetNodeSchedulersResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var NodeSchedulerServiceService = exports.NodeSchedulerServiceService = {
  getNodeSchedulers: {
    path: '/nodescheduler.NodeSchedulerService/GetNodeSchedulers',
    requestStream: false,
    responseStream: false,
    requestType: nodescheduler_pb.GetNodeSchedulersRequest,
    responseType: nodescheduler_pb.GetNodeSchedulersResponse,
    requestSerialize: serialize_nodescheduler_GetNodeSchedulersRequest,
    requestDeserialize: deserialize_nodescheduler_GetNodeSchedulersRequest,
    responseSerialize: serialize_nodescheduler_GetNodeSchedulersResponse,
    responseDeserialize: deserialize_nodescheduler_GetNodeSchedulersResponse,
  },
  getNodeScheduler: {
    path: '/nodescheduler.NodeSchedulerService/GetNodeScheduler',
    requestStream: false,
    responseStream: false,
    requestType: nodescheduler_pb.GetNodeSchedulerRequest,
    responseType: nodescheduler_pb.GetNodeSchedulerResponse,
    requestSerialize: serialize_nodescheduler_GetNodeSchedulerRequest,
    requestDeserialize: deserialize_nodescheduler_GetNodeSchedulerRequest,
    responseSerialize: serialize_nodescheduler_GetNodeSchedulerResponse,
    responseDeserialize: deserialize_nodescheduler_GetNodeSchedulerResponse,
  },
};

exports.NodeSchedulerServiceClient = grpc.makeGenericClientConstructor(NodeSchedulerServiceService, 'NodeSchedulerService');
