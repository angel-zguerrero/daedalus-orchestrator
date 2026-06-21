// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var tenant_pb = require('./tenant_pb.js');

function serialize_tenant_AssertBulkTenantRequest(arg) {
  if (!(arg instanceof tenant_pb.AssertBulkTenantRequest)) {
    throw new Error('Expected argument of type tenant.AssertBulkTenantRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_AssertBulkTenantRequest(buffer_arg) {
  return tenant_pb.AssertBulkTenantRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_AssertBulkTenantResponse(arg) {
  if (!(arg instanceof tenant_pb.AssertBulkTenantResponse)) {
    throw new Error('Expected argument of type tenant.AssertBulkTenantResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_AssertBulkTenantResponse(buffer_arg) {
  return tenant_pb.AssertBulkTenantResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_AssertTenantRequest(arg) {
  if (!(arg instanceof tenant_pb.AssertTenantRequest)) {
    throw new Error('Expected argument of type tenant.AssertTenantRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_AssertTenantRequest(buffer_arg) {
  return tenant_pb.AssertTenantRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_AssertTenantResponse(arg) {
  if (!(arg instanceof tenant_pb.AssertTenantResponse)) {
    throw new Error('Expected argument of type tenant.AssertTenantResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_AssertTenantResponse(buffer_arg) {
  return tenant_pb.AssertTenantResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_DeleteTenantRequest(arg) {
  if (!(arg instanceof tenant_pb.DeleteTenantRequest)) {
    throw new Error('Expected argument of type tenant.DeleteTenantRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_DeleteTenantRequest(buffer_arg) {
  return tenant_pb.DeleteTenantRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_DeleteTenantResponse(arg) {
  if (!(arg instanceof tenant_pb.DeleteTenantResponse)) {
    throw new Error('Expected argument of type tenant.DeleteTenantResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_DeleteTenantResponse(buffer_arg) {
  return tenant_pb.DeleteTenantResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_GetTenantsRequest(arg) {
  if (!(arg instanceof tenant_pb.GetTenantsRequest)) {
    throw new Error('Expected argument of type tenant.GetTenantsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_GetTenantsRequest(buffer_arg) {
  return tenant_pb.GetTenantsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_GetTenantsResponse(arg) {
  if (!(arg instanceof tenant_pb.GetTenantsResponse)) {
    throw new Error('Expected argument of type tenant.GetTenantsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_GetTenantsResponse(buffer_arg) {
  return tenant_pb.GetTenantsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_TenantInfoRequest(arg) {
  if (!(arg instanceof tenant_pb.TenantInfoRequest)) {
    throw new Error('Expected argument of type tenant.TenantInfoRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_TenantInfoRequest(buffer_arg) {
  return tenant_pb.TenantInfoRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_TenantInfoResponse(arg) {
  if (!(arg instanceof tenant_pb.TenantInfoResponse)) {
    throw new Error('Expected argument of type tenant.TenantInfoResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_TenantInfoResponse(buffer_arg) {
  return tenant_pb.TenantInfoResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_TenantSummaryRequest(arg) {
  if (!(arg instanceof tenant_pb.TenantSummaryRequest)) {
    throw new Error('Expected argument of type tenant.TenantSummaryRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_TenantSummaryRequest(buffer_arg) {
  return tenant_pb.TenantSummaryRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_tenant_TenantSummaryResponse(arg) {
  if (!(arg instanceof tenant_pb.TenantSummaryResponse)) {
    throw new Error('Expected argument of type tenant.TenantSummaryResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_tenant_TenantSummaryResponse(buffer_arg) {
  return tenant_pb.TenantSummaryResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var TenantServiceService = exports.TenantServiceService = {
  getTenantInfo: {
    path: '/tenant.TenantService/GetTenantInfo',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.TenantInfoRequest,
    responseType: tenant_pb.TenantInfoResponse,
    requestSerialize: serialize_tenant_TenantInfoRequest,
    requestDeserialize: deserialize_tenant_TenantInfoRequest,
    responseSerialize: serialize_tenant_TenantInfoResponse,
    responseDeserialize: deserialize_tenant_TenantInfoResponse,
  },
  getTenantSummary: {
    path: '/tenant.TenantService/GetTenantSummary',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.TenantSummaryRequest,
    responseType: tenant_pb.TenantSummaryResponse,
    requestSerialize: serialize_tenant_TenantSummaryRequest,
    requestDeserialize: deserialize_tenant_TenantSummaryRequest,
    responseSerialize: serialize_tenant_TenantSummaryResponse,
    responseDeserialize: deserialize_tenant_TenantSummaryResponse,
  },
  assertTenant: {
    path: '/tenant.TenantService/AssertTenant',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.AssertTenantRequest,
    responseType: tenant_pb.AssertTenantResponse,
    requestSerialize: serialize_tenant_AssertTenantRequest,
    requestDeserialize: deserialize_tenant_AssertTenantRequest,
    responseSerialize: serialize_tenant_AssertTenantResponse,
    responseDeserialize: deserialize_tenant_AssertTenantResponse,
  },
  assertBulkTenant: {
    path: '/tenant.TenantService/AssertBulkTenant',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.AssertBulkTenantRequest,
    responseType: tenant_pb.AssertBulkTenantResponse,
    requestSerialize: serialize_tenant_AssertBulkTenantRequest,
    requestDeserialize: deserialize_tenant_AssertBulkTenantRequest,
    responseSerialize: serialize_tenant_AssertBulkTenantResponse,
    responseDeserialize: deserialize_tenant_AssertBulkTenantResponse,
  },
  deleteTenant: {
    path: '/tenant.TenantService/DeleteTenant',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.DeleteTenantRequest,
    responseType: tenant_pb.DeleteTenantResponse,
    requestSerialize: serialize_tenant_DeleteTenantRequest,
    requestDeserialize: deserialize_tenant_DeleteTenantRequest,
    responseSerialize: serialize_tenant_DeleteTenantResponse,
    responseDeserialize: deserialize_tenant_DeleteTenantResponse,
  },
  getTenants: {
    path: '/tenant.TenantService/GetTenants',
    requestStream: false,
    responseStream: false,
    requestType: tenant_pb.GetTenantsRequest,
    responseType: tenant_pb.GetTenantsResponse,
    requestSerialize: serialize_tenant_GetTenantsRequest,
    requestDeserialize: deserialize_tenant_GetTenantsRequest,
    responseSerialize: serialize_tenant_GetTenantsResponse,
    responseDeserialize: deserialize_tenant_GetTenantsResponse,
  },
};

exports.TenantServiceClient = grpc.makeGenericClientConstructor(TenantServiceService, 'TenantService');
