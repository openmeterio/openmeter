# Fastify Example

In this example we will meter API calls with Fastify.
This is useful if you are building a web server and want to charge for certain API calls.

Language: Node.js, TypeScript

## Example

Fastify is a powerful Node.js web server that provides lifecycle hooks, useful to handle metering.
Checkout `app.ts` file and observe the flow of the request:

1. We set user on session
1. We serve `GET /api` request
1. We meter request after it's served

### Run The Example

Run the code as:

```sh
npm install
npm start
```

Visit the metered `http://localhost:3000/api` in your browser couple of times to generate usage.
Observe meter changes on `http://localhost:3000`. (should see updates in 1-2 seconds)

You should see your metered usage group by path, method in an hourly resolution:

```json
{
  "windowSize": "HOUR",
  "data": [
    {
      "subject": "my-test-id",
      "windowStart": "2023-07-01T20:00:00Z",
      "windowEnd": "2023-07-01T21:00:00Z",
      "value": 10.3178,
      "groupBy": {
        "method": "GET",
        "path": "/api"
      }
    },
    {
      "subject": "my-test-id",
      "windowStart": "2023-07-01T21:00:00Z",
      "windowEnd": "2023-07-01T22:00:00Z",
      "value": 15.818299999999999,
      "groupBy": {
        "method": "GET",
        "path": "/api"
      }
    }
  ]
}
```
