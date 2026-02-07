export const handler = async (event) => {
    const backendUrl = process.env.BACKEND_URL
    const connectionId = event.requestContext.connectionId
    const headers = {
        "Content-Type": "application/json",
        "X-Connection-Id": connectionId
    }

    const body = event.body || "{}"

    try {
        const response = await fetch(backendUrl, {
            method: "POST",
            headers: headers,
            body: body
        })

        return { statusCode: response.status }
    } catch (error) {
        console.error("Message Shim Error:", error)
        return {
            statusCode: 502,
            body: JSON.stringify({
                "message": "Bad Gateway: Backend Unreachable"
            })
        }
    }
}