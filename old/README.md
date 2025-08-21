## Getting Started

In the main directory, create a link to the environment file:
```bash
ln -s $PWD/deployments/local/.env ./web
```

Use the proper node version
```bash
nvm use
```

Install dependencies
```bash
npm install
```

Then run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.
