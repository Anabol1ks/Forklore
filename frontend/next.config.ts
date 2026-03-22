import type { NextConfig } from "next";

const nextConfig: NextConfig = {
	allowedDevOrigins: [
		'26.117.119.83',
		'*.lhr.life',
		'https://197ae779b628f4.lhr.life',
		'https://c2ui1i-83-69-253-9.ru.tuna.am',
		'https://qnne5e-83-69-253-9.ru.tuna.am',
		'*.ru.tuna.am',
	],
	turbopack: {
		root: __dirname,
	},
}

export default nextConfig;
