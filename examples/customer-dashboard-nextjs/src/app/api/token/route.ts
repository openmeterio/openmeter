import { OpenMeter } from '@openmeter/sdk';

export async function GET(request: Request) {
  // TODO: authenticate user, resolve subject
  const subject = process.env.NEXT_PUBLIC_OPENMETER_SUBJECT;

  const openmeter = new OpenMeter({
    baseUrl: process.env.NEXT_PUBLIC_OPENMETER_URL,
    token: process.env.OPENMETER_API_TOKEN,
  });
  const data = await openmeter.portal.createToken({ subject });

  return Response.json(data);
}
