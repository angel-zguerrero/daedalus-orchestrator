// package: auth
// file: auth.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as auth_pb from "./auth_pb";

interface IAuthServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    login: IAuthServiceService_ILogin;
    logout: IAuthServiceService_ILogout;
}

interface IAuthServiceService_ILogin extends grpc.MethodDefinition<auth_pb.LoginRequest, auth_pb.LoginResponse> {
    path: "/auth.AuthService/Login";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<auth_pb.LoginRequest>;
    requestDeserialize: grpc.deserialize<auth_pb.LoginRequest>;
    responseSerialize: grpc.serialize<auth_pb.LoginResponse>;
    responseDeserialize: grpc.deserialize<auth_pb.LoginResponse>;
}
interface IAuthServiceService_ILogout extends grpc.MethodDefinition<auth_pb.LogoutRequest, auth_pb.LogoutResponse> {
    path: "/auth.AuthService/Logout";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<auth_pb.LogoutRequest>;
    requestDeserialize: grpc.deserialize<auth_pb.LogoutRequest>;
    responseSerialize: grpc.serialize<auth_pb.LogoutResponse>;
    responseDeserialize: grpc.deserialize<auth_pb.LogoutResponse>;
}

export const AuthServiceService: IAuthServiceService;

export interface IAuthServiceServer extends grpc.UntypedServiceImplementation {
    login: grpc.handleUnaryCall<auth_pb.LoginRequest, auth_pb.LoginResponse>;
    logout: grpc.handleUnaryCall<auth_pb.LogoutRequest, auth_pb.LogoutResponse>;
}

export interface IAuthServiceClient {
    login(request: auth_pb.LoginRequest, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    login(request: auth_pb.LoginRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    login(request: auth_pb.LoginRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    logout(request: auth_pb.LogoutRequest, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
    logout(request: auth_pb.LogoutRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
    logout(request: auth_pb.LogoutRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
}

export class AuthServiceClient extends grpc.Client implements IAuthServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public login(request: auth_pb.LoginRequest, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    public login(request: auth_pb.LoginRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    public login(request: auth_pb.LoginRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: auth_pb.LoginResponse) => void): grpc.ClientUnaryCall;
    public logout(request: auth_pb.LogoutRequest, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
    public logout(request: auth_pb.LogoutRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
    public logout(request: auth_pb.LogoutRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: auth_pb.LogoutResponse) => void): grpc.ClientUnaryCall;
}
