import { MockAgent } from 'undici';

export const mockAgent = new MockAgent()
mockAgent.disableNetConnect()

const client = mockAgent.get('http://127.0.0.1:8888');
client.intercept({
    path: '/api/v1/events',
    method: 'POST',
    headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/cloudevents+json',
    },
    body: JSON.stringify({
        specversion: '1.0',
        id: 'id-1',
        source: 'my-app',
        type: 'my-type',
        subject: 'my-awesome-user-id',
        time: new Date('2023-01-01'),
        data: {
            api_calls: 1,
        },
    })
})
    .reply(201);
client.intercept({
    path: '/api/v1/events',
    method: 'POST',
    headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/cloudevents+json',
    },
    body: JSON.stringify({
        specversion: '1.0',
        id: 'aaf17be7-860c-4519-91d3-00d97da3cc65',
        source: '@openmeter/sdk',
        type: 'my-type',
        subject: 'my-awesome-user-id',
        data: {
            api_calls: 1,
        },
    })
})
    .reply(201)

client.intercept({
    path: '/api/v1/meters/m1/values',
    method: 'GET',
    headers: {
        'Accept': 'application/json'
    }
})
    .reply(200, {
        windowSize: 'HOUR',
        data: [
            {
                subject: 'customer-1',
                windowStart: '2023-01-01T01:00:00.001Z',
                windowEnd: '2023-01-01T01:00:00.001Z',
                value: 1,
                groupBy: {
                    method: 'GET'
                }
            }
        ]
    });
