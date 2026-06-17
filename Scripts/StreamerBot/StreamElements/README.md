# StreamElements Points Fetcher for Streamer.bot

Retrieves a user's StreamElements loyalty points and stores the value in a Streamer.bot variable for use in commands, actions, automated responses, and workflows.

## Features

* Retrieves user points through the StreamElements API.
* Fully compatible with Streamer.bot.
* Stores the result in the `%sepoints%` variable.
* Automatically detects:

  * `username`
  * `user`
  * `targetUser`
* Network and API error handling.
* Detailed logging inside Streamer.bot.

---

## Requirements

* Streamer.bot
* StreamElements account
* StreamElements JWT Token
* StreamElements Channel ID

---

## Configuration

Replace the following values inside the script:

```csharp
private readonly string _jwtToken = "YOUR_JWT_TOKEN";
private readonly string _channelId = "YOUR_CHANNEL_ID";
```

### Getting Your JWT Token

1. Log in to StreamElements.
2. Open Account Settings.
3. Generate or copy your JWT Token.

### Getting Your Channel ID

You can find it in your StreamElements dashboard URL or retrieve it through the API.

---

## Usage

Execute the action from Streamer.bot while providing one of the following variables:

* `username`
* `user`
* `targetUser`

The script will query StreamElements and store the result in:

```text
%sepoints%
```

### Example

If the user has 1250 loyalty points:

```text
%sepoints% = 1250
```

You can then use it in responses:

```text
@%user% has %sepoints% points.
```

---

## Generated Variables

| Variable     | Description                             |
| ------------ | --------------------------------------- |
| `%sepoints%` | User loyalty points from StreamElements |

---

## How It Works

1. Retrieves the username.
2. Sends a request to the StreamElements API.
3. Processes the JSON response.
4. Extracts the points value.
5. Stores the result in `%sepoints%`.
6. Logs information and errors inside Streamer.bot.

---

## Error Handling

If any issue occurs:

* User not found.
* Invalid token.
* Connection failure.
* Unexpected API response.

The script will return:

```text
0
```

and log the reason inside Streamer.bot.

---

## License

This project is released under the MIT License.
