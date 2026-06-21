// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var exchange_pb = require('./exchange_pb.js');

function serialize_exchange_BulkCreateExchangeRequest(arg) {
  if (!(arg instanceof exchange_pb.BulkCreateExchangeRequest)) {
    throw new Error('Expected argument of type exchange.BulkCreateExchangeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_BulkCreateExchangeRequest(buffer_arg) {
  return exchange_pb.BulkCreateExchangeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_BulkCreateExchangeResponse(arg) {
  if (!(arg instanceof exchange_pb.BulkCreateExchangeResponse)) {
    throw new Error('Expected argument of type exchange.BulkCreateExchangeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_BulkCreateExchangeResponse(buffer_arg) {
  return exchange_pb.BulkCreateExchangeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_CreateExchangeRequest(arg) {
  if (!(arg instanceof exchange_pb.CreateExchangeRequest)) {
    throw new Error('Expected argument of type exchange.CreateExchangeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_CreateExchangeRequest(buffer_arg) {
  return exchange_pb.CreateExchangeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_CreateExchangeResponse(arg) {
  if (!(arg instanceof exchange_pb.CreateExchangeResponse)) {
    throw new Error('Expected argument of type exchange.CreateExchangeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_CreateExchangeResponse(buffer_arg) {
  return exchange_pb.CreateExchangeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_DeleteExchangeRequest(arg) {
  if (!(arg instanceof exchange_pb.DeleteExchangeRequest)) {
    throw new Error('Expected argument of type exchange.DeleteExchangeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_DeleteExchangeRequest(buffer_arg) {
  return exchange_pb.DeleteExchangeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_DeleteExchangeResponse(arg) {
  if (!(arg instanceof exchange_pb.DeleteExchangeResponse)) {
    throw new Error('Expected argument of type exchange.DeleteExchangeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_DeleteExchangeResponse(buffer_arg) {
  return exchange_pb.DeleteExchangeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_GetExchangeRequest(arg) {
  if (!(arg instanceof exchange_pb.GetExchangeRequest)) {
    throw new Error('Expected argument of type exchange.GetExchangeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_GetExchangeRequest(buffer_arg) {
  return exchange_pb.GetExchangeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_GetExchangeResponse(arg) {
  if (!(arg instanceof exchange_pb.GetExchangeResponse)) {
    throw new Error('Expected argument of type exchange.GetExchangeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_GetExchangeResponse(buffer_arg) {
  return exchange_pb.GetExchangeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_GetExchangesRequest(arg) {
  if (!(arg instanceof exchange_pb.GetExchangesRequest)) {
    throw new Error('Expected argument of type exchange.GetExchangesRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_GetExchangesRequest(buffer_arg) {
  return exchange_pb.GetExchangesRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_GetExchangesResponse(arg) {
  if (!(arg instanceof exchange_pb.GetExchangesResponse)) {
    throw new Error('Expected argument of type exchange.GetExchangesResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_GetExchangesResponse(buffer_arg) {
  return exchange_pb.GetExchangesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_PublishMessageRequest(arg) {
  if (!(arg instanceof exchange_pb.PublishMessageRequest)) {
    throw new Error('Expected argument of type exchange.PublishMessageRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_PublishMessageRequest(buffer_arg) {
  return exchange_pb.PublishMessageRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_exchange_PublishMessageResponse(arg) {
  if (!(arg instanceof exchange_pb.PublishMessageResponse)) {
    throw new Error('Expected argument of type exchange.PublishMessageResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_exchange_PublishMessageResponse(buffer_arg) {
  return exchange_pb.PublishMessageResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var ExchangeServiceService = exports.ExchangeServiceService = {
  createExchange: {
    path: '/exchange.ExchangeService/CreateExchange',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.CreateExchangeRequest,
    responseType: exchange_pb.CreateExchangeResponse,
    requestSerialize: serialize_exchange_CreateExchangeRequest,
    requestDeserialize: deserialize_exchange_CreateExchangeRequest,
    responseSerialize: serialize_exchange_CreateExchangeResponse,
    responseDeserialize: deserialize_exchange_CreateExchangeResponse,
  },
  bulkCreateExchange: {
    path: '/exchange.ExchangeService/BulkCreateExchange',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.BulkCreateExchangeRequest,
    responseType: exchange_pb.BulkCreateExchangeResponse,
    requestSerialize: serialize_exchange_BulkCreateExchangeRequest,
    requestDeserialize: deserialize_exchange_BulkCreateExchangeRequest,
    responseSerialize: serialize_exchange_BulkCreateExchangeResponse,
    responseDeserialize: deserialize_exchange_BulkCreateExchangeResponse,
  },
  getExchange: {
    path: '/exchange.ExchangeService/GetExchange',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.GetExchangeRequest,
    responseType: exchange_pb.GetExchangeResponse,
    requestSerialize: serialize_exchange_GetExchangeRequest,
    requestDeserialize: deserialize_exchange_GetExchangeRequest,
    responseSerialize: serialize_exchange_GetExchangeResponse,
    responseDeserialize: deserialize_exchange_GetExchangeResponse,
  },
  getExchanges: {
    path: '/exchange.ExchangeService/GetExchanges',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.GetExchangesRequest,
    responseType: exchange_pb.GetExchangesResponse,
    requestSerialize: serialize_exchange_GetExchangesRequest,
    requestDeserialize: deserialize_exchange_GetExchangesRequest,
    responseSerialize: serialize_exchange_GetExchangesResponse,
    responseDeserialize: deserialize_exchange_GetExchangesResponse,
  },
  deleteExchange: {
    path: '/exchange.ExchangeService/DeleteExchange',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.DeleteExchangeRequest,
    responseType: exchange_pb.DeleteExchangeResponse,
    requestSerialize: serialize_exchange_DeleteExchangeRequest,
    requestDeserialize: deserialize_exchange_DeleteExchangeRequest,
    responseSerialize: serialize_exchange_DeleteExchangeResponse,
    responseDeserialize: deserialize_exchange_DeleteExchangeResponse,
  },
  publishMessage: {
    path: '/exchange.ExchangeService/PublishMessage',
    requestStream: false,
    responseStream: false,
    requestType: exchange_pb.PublishMessageRequest,
    responseType: exchange_pb.PublishMessageResponse,
    requestSerialize: serialize_exchange_PublishMessageRequest,
    requestDeserialize: deserialize_exchange_PublishMessageRequest,
    responseSerialize: serialize_exchange_PublishMessageResponse,
    responseDeserialize: deserialize_exchange_PublishMessageResponse,
  },
};

exports.ExchangeServiceClient = grpc.makeGenericClientConstructor(ExchangeServiceService, 'ExchangeService');
