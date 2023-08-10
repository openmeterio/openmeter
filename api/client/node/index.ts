import { NodeHttpRequest } from './generated/core/NodeHttpRequest.js'
export { NodeHttpRequest } from './generated/core/NodeHttpRequest.js'
import { HttpService } from './generated/HttpService.js'
import {
	DefaultService,
	EventsService,
	MetersService,
} from './generated/index.js'
import type { BaseHttpRequest } from './generated/core/BaseHttpRequest.js'
import type { OpenAPIConfig } from './generated/core/OpenAPI.js'
export * from './generated/index.js'

export type HttpRequestConstructor = new (
	config: OpenAPIConfig
) => BaseHttpRequest

export type ClientConfig = {
	baseUrl: string
	credentials?: OpenAPIConfig['CREDENTIALS']
	withCredentials?: OpenAPIConfig['WITH_CREDENTIALS']
	token?: OpenAPIConfig['TOKEN']
	username?: OpenAPIConfig['USERNAME']
	password?: OpenAPIConfig['PASSWORD']
	headers?: OpenAPIConfig['HEADERS']
}

export class OpenMeter extends HttpService {
	public readonly request: BaseHttpRequest
	public readonly default: DefaultService
	public readonly events: EventsService
	public readonly meters: MetersService

	constructor(
		config: ClientConfig,
		HttpRequest: HttpRequestConstructor = NodeHttpRequest
	) {
		const openAPIConfig: OpenAPIConfig = {
			BASE: config.baseUrl,
			VERSION: '1.0.0',
			CREDENTIALS: config.credentials ?? 'include',
			WITH_CREDENTIALS: config.withCredentials ?? false,
			TOKEN: config.token,
			USERNAME: config.username,
			PASSWORD: config.password,
			HEADERS: config.headers,
		}
		super(openAPIConfig, HttpRequest)
		this.request = new HttpRequest(openAPIConfig)
		this.default = new DefaultService(this.request)
		this.events = new EventsService(this.request)
		this.meters = new MetersService(this.request)
	}
}
