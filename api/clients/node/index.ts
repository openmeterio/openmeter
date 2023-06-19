import { NodeHttpRequest } from './generated/core/NodeHttpRequest.js'
import { DefaultService } from './generated/services/DefaultService.js'
import type { BaseHttpRequest } from './generated/core/BaseHttpRequest.js'
import type { OpenAPIConfig } from './generated/core/OpenAPI.js'
export * from './generated/index.js'

export type HttpRequestConstructor = new (
	config: OpenAPIConfig
) => BaseHttpRequest

export type ClientConfig = {
	baseUrl: string
}

export class OpenMeter extends DefaultService {
	constructor(
		config: ClientConfig,
		HttpRequest: HttpRequestConstructor = NodeHttpRequest
	) {
		const openAPIConfig: OpenAPIConfig = {
			BASE: config.baseUrl,
			VERSION: '1.0.0',
			CREDENTIALS: 'include',
			WITH_CREDENTIALS: false,
		}
		const request = new HttpRequest(openAPIConfig)

		super(request)
	}
}
