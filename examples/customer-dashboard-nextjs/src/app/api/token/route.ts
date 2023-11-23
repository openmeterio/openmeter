import { OpenMeter } from '@openmeter/sdk'

export async function GET(request: Request) {
  // TODO: authenticate user, resolve subject
  const subject = process.env.OPENMETER_SUBJECT

  const openmeter = new OpenMeter({
    baseUrl: process.env.OPENMETER_URL,
  })
  const data = await openmeter.portal.createToken({ subject })

  return Response.json(data)
}
