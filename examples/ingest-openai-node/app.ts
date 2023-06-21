import assert from 'assert'

import 'dotenv/config'
import { OpenMeter } from '@openmeter/sdk'
import { Configuration, OpenAIApi } from 'openai'

// Environment variables
assert.ok(
    process.env.OPENAI_ORG,
    'OPENAI_ORG environment variables is required'
)
assert.ok(
    process.env.OPENAI_API_KEY,
    'OPENAI_API_KEY environment variables is required'
)

const configuration = new Configuration({
    organization: process.env.OPENAI_ORG,
    apiKey: process.env.OPENAI_API_KEY,
})
const openai = new OpenAIApi(configuration)
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

async function main() {
    const { data } = await openai.createChatCompletion({
        model: 'gpt-3.5-turbo',
        messages: [{ role: 'user', content: 'Hello world' }],
    })

    if (data.usage) {
        await openmeter.ingestEvents({
            specversion: '1.0',
            // We use Open AI response ID as idempotent key
            id: data.id,
            source: 'my-app',
            type: 'openai',
            subject: 'my-awesome-user-id',
            // We get date from Open AI response
            time: new Date(data.created * 1000).toISOString(),
            // We report usage with model
            data: {
                total_tokens: data.usage.total_tokens,
                prompt_tokens: data.usage.prompt_tokens,
                completion_tokens: data.usage.completion_tokens,
                model: data.model,
            },
        })

        // Debug logs
        console.debug(
            `input: Hello world, output: ${data.choices
                .map(({ message }) => message?.content)
                .join(' ')}`
        )
        console.debug(
            `total_tokens: ${data.usage.total_tokens}, subject: my-awesome-user-id`
        )
    }
}

main()
    .then(() => console.log('done'))
    .catch((err) => console.error(err))
