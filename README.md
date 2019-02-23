# Facebook Webhook as a Service


## Usage

- Starts the forward daemon locally

    ```
    $ ./forward http://localhost:4000/facebook/webhook_callback
    Forwarding SSE from "https://fbwhs.herokuapp.com/webhook/1HbA4TRlBeiS1nrfu5siRdgma7c" to "http://localhost:4000/facebook/webhook_callback"
    Usage:
    curl -X POST -d 'test=123' "https://fbwhs.herokuapp.com/webhook/1HbA4TRlBeiS1nrfu5siRdgma7c"
    ```

- Registers the webhook callback from above with Facebook

    ```
    curl -F "object=user" \
        -F "callback_url=https://fbwhs.herokuapp.com/webhook/1HbA4TRlBeiS1nrfu5siRdgma7c" \
        -F "fields=about" \
        -F "verify_token=1HbA4TRlBeiS1nrfu5siRdgma7c" \
        -F "access_token={FB_APP_ACCESS_TOKEN}" \
        "https://graph.facebook.com/{FB_APP_ID}/subscriptions"
    ```

    Note: the last part of the webhook address is the `verify_token`. E.g. `https://fbwhs.herokuapp.com/webhook/abc123` where `abc123` is the `verify_token`.

- Now, whenever Facebook sends a webhook to `https://fbwhs.herokuapp.com/webhook/1HbA4TRlBeiS1nrfu5siRdgma7c`, the daemon running locally will forward the request to your local server.

- Also, you can test it with `curl -X POST -d 'test=123' "https://fbwhs.herokuapp.com/webhook/1HbA4TRlBeiS1nrfu5siRdgma7c"`, the local server callback `http://localhost:4000/facebook/webhook_callback` should recieve a request with a body `test=123`.


## How it works

tldr; The concept is same as [smee](https://smee.io/), but we handle Facebook's [verification request](https://developers.facebook.com/docs/graph-api/webhooks/getting-started#verification-requests) for you.
