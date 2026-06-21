// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var metrics_pb = require('./metrics_pb.js');

function serialize_metrics_SystemMetricsRequest(arg) {
  if (!(arg instanceof metrics_pb.SystemMetricsRequest)) {
    throw new Error('Expected argument of type metrics.SystemMetricsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_metrics_SystemMetricsRequest(buffer_arg) {
  return metrics_pb.SystemMetricsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_metrics_SystemMetricsResponse(arg) {
  if (!(arg instanceof metrics_pb.SystemMetricsResponse)) {
    throw new Error('Expected argument of type metrics.SystemMetricsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_metrics_SystemMetricsResponse(buffer_arg) {
  return metrics_pb.SystemMetricsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var MetricsServiceService = exports.MetricsServiceService = {
  getSystemMetrics: {
    path: '/metrics.MetricsService/GetSystemMetrics',
    requestStream: false,
    responseStream: false,
    requestType: metrics_pb.SystemMetricsRequest,
    responseType: metrics_pb.SystemMetricsResponse,
    requestSerialize: serialize_metrics_SystemMetricsRequest,
    requestDeserialize: deserialize_metrics_SystemMetricsRequest,
    responseSerialize: serialize_metrics_SystemMetricsResponse,
    responseDeserialize: deserialize_metrics_SystemMetricsResponse,
  },
};

exports.MetricsServiceClient = grpc.makeGenericClientConstructor(MetricsServiceService, 'MetricsService');
