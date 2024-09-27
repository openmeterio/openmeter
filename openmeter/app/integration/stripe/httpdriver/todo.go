package httpdriver

/*
Each integration can provide it's own endpoints (e.g. we define the endpoints for stripe in the
openapi schema, then the main router resolves the app and calls the appropriate handler defined
here).

URLs for the stripe integration (<base_url> is /api/v1/apps/<app_type>/applications/<app_id>):

<base_url>/settings
  GET: get current settings such as account_id, supplier information, etc.
  POST: update settings such as account_id, supplier information, etc.

<base_url>/connect
	OAUTH endpoint

<base_url>/customers/<customer_id>
	GET: get customer specific overries
	POST: update customer specific overrides
	DELETE: delete customer specific overrides

The integration has it's own adapter that is resposible for persisting the data
*/
