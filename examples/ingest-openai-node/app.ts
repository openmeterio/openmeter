import { Configuration, OpenAIApi } from "openai";
import { OpenAPIClientAxios } from 'openapi-client-axios'
import { Client as OpenMeterClient } from './openapi';

const configuration = new Configuration({
  organization: process.env.OPENAI_ORG,
  apiKey: process.env.OPENAI_API_KEY,
});
const openai = new OpenAIApi(configuration);
const openmeter = new OpenAPIClientAxios({ definition: 'https://openmeter.io/api/openapi.json' }).initSync<OpenMeterClient>();

async function main() {
  const { data } = await openai.createChatCompletion({
    model: "gpt-3.5-turbo",
    messages: [{ role: "user", content: "Hello world" }],
  });

  if (data.usage) {
    await openmeter.ingestEvents(null, {
      specversion: '1.0',
      // We use Open AI response ID as idempotent key
      id: data.id,
      source: 'my-app',
      type: 'openai',
      subject: 'my-awesome-user-id',
      // We use Open AI response dat as event date
      time: new Date(data.created * 1000).toISOString(),
      data: {
        total_tokens: data.usage.total_tokens,
        prompt_tokens: data.usage.prompt_tokens,
        completion_tokens: data.usage.completion_tokens,
        model: data.model
      }
    })
  }
}

main()
  .then(() => console.log('done'))
  .catch((err) => console.error(err))