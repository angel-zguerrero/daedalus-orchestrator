// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var jobworker_pb = require('./jobworker_pb.js');

function serialize_jobworker_AckMessageRequest(arg) {
  if (!(arg instanceof jobworker_pb.AckMessageRequest)) {
    throw new Error('Expected argument of type jobworker.AckMessageRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_jobworker_AckMessageRequest(buffer_arg) {
  return jobworker_pb.AckMessageRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_jobworker_AckMessageResponse(arg) {
  if (!(arg instanceof jobworker_pb.AckMessageResponse)) {
    throw new Error('Expected argument of type jobworker.AckMessageResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_jobworker_AckMessageResponse(buffer_arg) {
  return jobworker_pb.AckMessageResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_jobworker_ClaimWorkRequest(arg) {
  if (!(arg instanceof jobworker_pb.ClaimWorkRequest)) {
    throw new Error('Expected argument of type jobworker.ClaimWorkRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_jobworker_ClaimWorkRequest(buffer_arg) {
  return jobworker_pb.ClaimWorkRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_jobworker_ClaimWorkStreamMessage(arg) {
  if (!(arg instanceof jobworker_pb.ClaimWorkStreamMessage)) {
    throw new Error('Expected argument of type jobworker.ClaimWorkStreamMessage');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_jobworker_ClaimWorkStreamMessage(buffer_arg) {
  return jobworker_pb.ClaimWorkStreamMessage.deserializeBinary(new Uint8Array(buffer_arg));
}


var JobWorkerServiceService = exports.JobWorkerServiceService = {
  claimWork: {
    path: '/jobworker.JobWorkerService/ClaimWork',
    requestStream: true,
    responseStream: true,
    requestType: jobworker_pb.ClaimWorkRequest,
    responseType: jobworker_pb.ClaimWorkStreamMessage,
    requestSerialize: serialize_jobworker_ClaimWorkRequest,
    requestDeserialize: deserialize_jobworker_ClaimWorkRequest,
    responseSerialize: serialize_jobworker_ClaimWorkStreamMessage,
    responseDeserialize: deserialize_jobworker_ClaimWorkStreamMessage,
  },
  ackMessage: {
    path: '/jobworker.JobWorkerService/AckMessage',
    requestStream: false,
    responseStream: false,
    requestType: jobworker_pb.AckMessageRequest,
    responseType: jobworker_pb.AckMessageResponse,
    requestSerialize: serialize_jobworker_AckMessageRequest,
    requestDeserialize: deserialize_jobworker_AckMessageRequest,
    responseSerialize: serialize_jobworker_AckMessageResponse,
    responseDeserialize: deserialize_jobworker_AckMessageResponse,
  },
};

exports.JobWorkerServiceClient = grpc.makeGenericClientConstructor(JobWorkerServiceService, 'JobWorkerService');
