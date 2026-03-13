// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';

export default defineConfig({
	site: 'https://zach-snell.github.io',
	base: '/jtk',
	integrations: [
		starlight({
			title: 'jtk',
			description: 'A dual-mode Go CLI & MCP Server for Jira Cloud',
			plugins: [
				starlightLlmsTxt({
					projectName: 'jtk (Jira Toolkit)',
					description: 'A dual-mode Go CLI and MCP Server for Jira Cloud. Provides 11 MCP tools with 50+ actions for issues, search, boards, sprints, projects, dev info, worklogs, versions, attachments, users, and metrics. Includes 4 MCP prompts for standup reports, sprint status, release notes, and dev tree analysis. Features git branch detection, dynamic permission introspection, markdown-to-ADF conversion, response flattening, and rate limiting.',
					customSets: [
						{
							label: 'MCP Tools',
							description: 'All MCP tool reference documentation',
							paths: ['mcp/**'],
						},
						{
							label: 'CLI',
							description: 'CLI command reference',
							paths: ['cli/**'],
						},
					],
				}),
			],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/zach-snell/jtk' },
			],
			editLink: {
				baseUrl: 'https://github.com/zach-snell/jtk/edit/main/docs/',
			},
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Configuration', slug: 'getting-started/configuration' },
						{ label: 'Quick Start', slug: 'getting-started/quickstart' },
					],
				},
				{
			label: 'CLI Commands',
				items: [
					{ label: 'Overview', slug: 'cli/overview' },
					{ label: 'jtk auth', slug: 'cli/auth' },
					{ label: 'jtk mcp', slug: 'cli/mcp' },
					{ label: 'jtk issues', slug: 'cli/issues' },
						{ label: 'jtk boards', slug: 'cli/boards' },
						{ label: 'jtk projects', slug: 'cli/projects' },
						{ label: 'jtk users', slug: 'cli/users' },
						{ label: 'jtk worklogs', slug: 'cli/worklogs' },
						{ label: 'jtk versions', slug: 'cli/versions' },
						{ label: 'jtk devinfo', slug: 'cli/devinfo' },
						{ label: 'jtk attachments', slug: 'cli/attachments' },
						{ label: 'jtk metrics', slug: 'cli/metrics' },
					],
				},
				{
					label: 'MCP Tool Reference',
					items: [
						{ label: 'Overview', slug: 'mcp/overview' },
						{ label: 'manage_issues', slug: 'mcp/manage-issues' },
						{ label: 'manage_search', slug: 'mcp/manage-search' },
						{ label: 'manage_boards', slug: 'mcp/manage-boards' },
						{ label: 'manage_projects', slug: 'mcp/manage-projects' },
						{ label: 'manage_devinfo', slug: 'mcp/manage-devinfo' },
						{ label: 'manage_worklogs', slug: 'mcp/manage-worklogs' },
						{ label: 'manage_versions', slug: 'mcp/manage-versions' },
						{ label: 'manage_attachments', slug: 'mcp/manage-attachments' },
					{ label: 'manage_users', slug: 'mcp/manage-users' },
					{ label: 'manage_metrics', slug: 'mcp/manage-metrics' },
					{ label: 'Prompts', slug: 'mcp/prompts' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Usage Examples', slug: 'guides/examples' },
						{ label: 'Git Integration', slug: 'guides/git-integration' },
						{ label: 'JQL Guide', slug: 'guides/jql-guide' },
					],
				},
				{
					label: 'Advanced',
					items: [
						{ label: 'Architecture', slug: 'advanced/architecture' },
						{ label: 'Security', slug: 'advanced/security' },
						{ label: 'Docker Deployment', slug: 'advanced/docker' },
						{ label: 'Development', slug: 'advanced/development' },
					],
				},
			],
		}),
	],
});
