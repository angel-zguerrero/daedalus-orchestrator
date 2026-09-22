import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface OAuthApp {
  id: string;
  clientId: string;
  name: string;
  tenantId: string;
  description: string;
  allowedScopes: string[];
  createdAt: string;
  updatedAt: string;
}

export interface OAuthAppCreated extends OAuthApp {
  clientSecret: string;
}

export interface CreateOAuthAppRequest {
  name: string;
  description: string;
  allowedScopes: string[];
}

@Injectable({ providedIn: 'root' })
export class OAuthAppsService {
  constructor(private http: HttpClient) {}

  list(tenantCode: string): Observable<{ items: OAuthApp[]; cursor: string }> {
    return this.http.get<{ items: OAuthApp[]; cursor: string }>(
      `/rest-api/tenants/${tenantCode}/oauth-apps`
    );
  }

  create(tenantCode: string, payload: CreateOAuthAppRequest): Observable<OAuthAppCreated> {
    return this.http.post<OAuthAppCreated>(
      `/rest-api/tenants/${tenantCode}/oauth-apps`,
      payload
    );
  }

  getById(tenantCode: string, appId: string): Observable<OAuthApp> {
    return this.http.get<OAuthApp>(
      `/rest-api/tenants/${tenantCode}/oauth-apps/${appId}`
    );
  }

  delete(tenantCode: string, appId: string): Observable<any> {
    return this.http.delete(
      `/rest-api/tenants/${tenantCode}/oauth-apps/${appId}`
    );
  }

  rotateSecret(tenantCode: string, appId: string): Observable<OAuthAppCreated> {
    return this.http.post<OAuthAppCreated>(
      `/rest-api/tenants/${tenantCode}/oauth-apps/${appId}/rotate-secret`,
      {}
    );
  }

  // Global Service Accounts
  listGlobal(): Observable<{ items: OAuthApp[]; cursor: string }> {
    return this.http.get<{ items: OAuthApp[]; cursor: string }>(
      '/rest-api/oauth-apps'
    );
  }

  createGlobal(payload: CreateOAuthAppRequest): Observable<OAuthAppCreated> {
    return this.http.post<OAuthAppCreated>('/rest-api/oauth-apps', payload);
  }

  getByIdGlobal(appId: string): Observable<OAuthApp> {
    return this.http.get<OAuthApp>(`/rest-api/oauth-apps/${appId}`);
  }

  deleteGlobal(appId: string): Observable<any> {
    return this.http.delete(`/rest-api/oauth-apps/${appId}`);
  }

  rotateSecretGlobal(appId: string): Observable<OAuthAppCreated> {
    return this.http.post<OAuthAppCreated>(
      `/rest-api/oauth-apps/${appId}/rotate-secret`,
      {}
    );
  }
}
