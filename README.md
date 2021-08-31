# tasker

> My task list, maybe will be a web service.
> I wanted to try out some things, and decided to make my own task manager.


## Features

- No javaScript
- Simple HTML with clean interface
- Uses Deta
- Free for you


## Hosting

You can host this in your own infrastructure, the final application is a
docker image, so one can change the deployment stage. Here I used Google
Cloud Run. You must provide only some environment variables:

- DETA_KEY: your deta key
- PORT: optional port, defaults to 8080
- EMAIL_FROM: the email to send mail to users
- EMAIL_PASS: your gmail app password

This project uses Gmail for sending email, contributions to support
other providers are welcome.


## License

MIT License, I take no responsibility for any damage caused by this software,
use at your own risk. Feel free to contribute, fork, clone or distribute. See
[LICENSE](https://github.com/blmayer/tasker/blob/main/LICENSE) for details.


## TODO:

- Let user provide a key for encryption
