// Defaults used by @mattermost/playwright-lib, which reads process.env while its
// module is evaluated. Import this before that package.
process.env.PW_BASE_URL = process.env.PW_BASE_URL || 'http://mattermost:8065';
process.env.PW_HEADLESS = process.env.PW_HEADLESS || 'true';
