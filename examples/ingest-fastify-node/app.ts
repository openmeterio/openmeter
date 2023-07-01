import { randomUUID } from 'crypto'
import { OpenMeter, WindowSize } from '@openmeter/sdk'
import fastify, { FastifyRequest } from 'fastify'
import fastifyCookie from '@fastify/cookie'
import fastifySession from '@fastify/session'

const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })
const server = fastify({
    logger: true
})

server.register(fastifyCookie);
server.register(fastifySession, { secret: 'a secret with minimum length of 32 characters' });

// To make testing this example easier we list all the meter values for root request
// This endpoint is not metered
server.get('/', {
    schema: {
        querystring: {
            subject: { type: 'string', nullable: true },
            from: { type: 'string', format: 'date-time', nullable: true },
            to: { type: 'string', format: 'date-time', nullable: true }
        },
    },
    handler: async (req: FastifyRequest<{ Querystring: { subject?: string, from?: string, to?: string } }>) => {
        const values = await openmeter.getValuesByMeterId('m1', req.query.subject, req.query.from, req.query.to, WindowSize.HOUR)
        return values
    }
})

// Metered APIs on /api
server.register((instance, opts, next) => {
    // Example metered API on GET /api
    instance.get('/', () => {
        return 'hello metered api'
    })

    // Set session, see: https://github.com/fastify/session
    instance.addHook('preHandler', (request, reply, next) => {
        request.session.user = { id: 'my-test-id', name: 'Test User' };
        next();
    })

    // Execute metering
    instance.addHook('onResponse', async (request, reply) => {
        const reqId = request.headers['x-request-id']
        const id = typeof reqId === 'string' ? reqId : randomUUID()

        await openmeter.ingestEvents({
            specversion: '1.0',
            id,
            source: 'my-app',
            type: 'api-calls',
            subject: request.session.user.id,
            time: new Date().toISOString(),
            data: {
                method: request.method,
                path: request.routerPath,
                duration_ms: reply.getResponseTime().toString(),
            },
        })
    })

    next()
}, { prefix: 'api' })

server.listen({ port: 3000 }, (err) => {
    if (err) {
        console.error(err)
        process.exit(1)
    }
})

declare module "fastify" {
    interface Session {
        user: {
            id: string
            name: string
        }
    }
}
