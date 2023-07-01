# Fastify Example

In this example we will meter API call with Fastify.
This is useful if you are building a web server and want to charge for certain API calls.

Language: Node.js, TypeScript

## Example

Fastify is a powerful Node.js web server which provides lifecycle hooks, useful to handle metering.
Checkout `app.ts` file and observe the flow of the request:

1. We set user on session
1. We server `GET /api` request
1. We meter request after it's served

Run the code as:

```sh
npm install
npm start
```
