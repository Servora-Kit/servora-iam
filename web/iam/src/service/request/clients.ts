import { createRequestHandler } from './requestHandler'
import type { RequestHandlerOptions } from './requestHandler'

import {
  createApplicationServiceClient,
  createAuthnServiceClient,
  createOrganizationServiceClient,
  createProjectServiceClient,
  createUserServiceClient,
} from '@servora/api-client/iam/service/v1/index'
import type {
  ApplicationService,
  AuthnService,
  OrganizationService,
  ProjectService,
  UserService,
} from '@servora/api-client/iam/service/v1/index'

export interface IamClients {
  authn: AuthnService
  user: UserService
  organization: OrganizationService
  project: ProjectService
  application: ApplicationService
}

export function createIamClients(
  options: RequestHandlerOptions = {},
): IamClients {
  const handler = createRequestHandler(options)

  return {
    authn: createAuthnServiceClient(handler),
    user: createUserServiceClient(handler),
    organization: createOrganizationServiceClient(handler),
    project: createProjectServiceClient(handler),
    application: createApplicationServiceClient(handler),
  }
}

export type { RequestHandlerOptions } from './requestHandler'
export { ApiError } from './requestHandler'
export type { ApiErrorKind, TokenStore, RequestHandler } from './requestHandler'
