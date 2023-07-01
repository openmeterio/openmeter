import { randomUUID } from 'crypto'
import { OpenMeter } from '@openmeter/sdk'
import fastify from 'fastify'
import fastifyCookie from '@fastify/cookie'
import fastifySession from '@fastify/session'

const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })
const server = fastify({
    logger: true
})

server.register(fastifyCookie);
server.register(fastifySession, { secret: 'a secret with minimum length of 32 characters' });

server.get('/', async () => {
    return 'hello root'
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
            type: 'request',
            subject: request.session.user.id,
            time: new Date().toISOString(),
            data: {
                method: request.method,
                path: request.routerPath,
                response_time: reply.getResponseTime().toString(),
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
