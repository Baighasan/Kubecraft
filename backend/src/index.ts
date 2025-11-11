import express from 'express';
import { server } from "./config"
import { testConnection } from './config/database';

const app = express();
const port = server.port;

async function startServer() {
    await testConnection();
    app.listen(port, () => console.log(`Server started, listening on port ${port}`))
}

startServer().catch(error => {
    console.error('Failed to start: ', error);
    process.exit(1);
})