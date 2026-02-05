import axios from "axios";

function resolveApiBaseURL() {
	let base = __API_URL__;

	if (typeof window === "undefined") {
		return base;
	}

	try {
		const parsed = new URL(base);
		const runningOnLocalhost = ["localhost", "127.0.0.1"].includes(window.location.hostname);
		if (parsed.hostname === "localhost" && !runningOnLocalhost) {
			parsed.hostname = window.location.hostname;
			base = parsed.toString();
		}
	} catch (_) {
		// keep __API_URL__ when malformed/unexpected
	}

	return base;
}

const instance = axios.create({
	baseURL: resolveApiBaseURL(),
	timeout: 1000 * 5
});

export default instance;
