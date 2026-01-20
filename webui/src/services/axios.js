import axios from "axios";

const instance = axios.create({
	baseURL: "http://localhost:3000", // Punta al tuo backend Go
	timeout: 1000 * 5
});

export default instance;