# frozen_string_literal: true

# Idempotent GitLab fixture for the Mattermost plugin E2E suite.
# Creates a trusted OAuth application, the gitlab-e2e user, an admin token,
# and the settings that let webhooks reach the Compose network.
# Groups and projects are created later by the specs.

APP_NAME = 'Mattermost GitLab Plugin E2E'
REDIRECT_URI = 'http://mattermost:8065/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete'
E2E_USERNAME = 'gitlab-e2e'
E2E_EMAIL = 'gitlab-e2e@example.com'
# GitLab rejects weak passwords, including any password that contains the
# user's name, username, email, or a predictable word such as "password".
E2E_PASSWORD = 'Ht6-Yr8c-Bn3q-W5'
E2E_NAME = 'GitLab E2E'
TOKEN_NAME = 'e2e-admin'
RUNNER_NAME = 'e2e-runner'
SCOPES = 'api read_user'

def organization
    if Organizations::Organization.respond_to?(:default_organization)
        Organizations::Organization.default_organization
    else
        Organizations::Organization.first
    end
end

def ensure_local_webhooks(settings)
    settings.allow_local_requests_from_web_hooks_and_services = true
    if settings.respond_to?(:dns_rebinding_protection_enabled=)
        settings.dns_rebinding_protection_enabled = false
    end
    settings.save!
end

def ensure_user(admin, org)
    user = User.find_by_username(E2E_USERNAME)
    if user.nil?
        result = Users::CreateService.new(admin, {
            username: E2E_USERNAME,
            email: E2E_EMAIL,
            name: E2E_NAME,
            password: E2E_PASSWORD,
            password_confirmation: E2E_PASSWORD,
            skip_confirmation: true,
            organization_id: org.id,
            admin: false,
        }).execute
        raise "failed to create #{E2E_USERNAME}: #{result.message}" unless result.success?

        user = result.payload[:user]
    else
        user.password = E2E_PASSWORD
        user.password_confirmation = E2E_PASSWORD
        user.skip_reconfirmation!
        user.confirmed_at ||= Time.current
        user.save!
    end
    raise "failed to save #{E2E_USERNAME}: #{user.errors.full_messages.to_sentence}" unless user.persisted?

    user
end

def ensure_token(admin, target, org)
    target.personal_access_tokens.where(name: TOKEN_NAME).find_each do |existing|
        existing.update!(revoked: true)
    end

    result = PersonalAccessTokens::CreateService.new(
        current_user: admin,
        target_user: target,
        organization_id: org.id,
        params: {
            name: TOKEN_NAME,
            scopes: ['api'],
            expires_at: 30.days.from_now.to_date,
        },
    ).execute
    raise "failed to create token for #{target.username}: #{result.message}" unless result.success?

    raw = result.payload[:personal_access_token].token
    raise "token for #{target.username} has no readable plaintext value" if raw.blank?

    raw
end

def ensure_runner(admin)
    Ci::Runner.where(description: RUNNER_NAME).find_each(&:destroy!)

    result = Ci::Runners::CreateRunnerService.new(
        user: admin,
        params: {runner_type: 'instance_type', description: RUNNER_NAME, run_untagged: true},
    ).execute
    raise "failed to create runner: #{result.message}" unless result.success?

    result.payload[:runner].token
end

def ensure_oauth_application(org)
    Authn::OauthApplication.where(name: APP_NAME).find_each(&:destroy!)

    app = Authn::OauthApplication.new(
        name: APP_NAME,
        redirect_uri: REDIRECT_URI,
        scopes: SCOPES,
        confidential: true,
        trusted: true,
        organization_id: org.id,
    )
    app.save!
    raise "failed to create OAuth application: #{app.errors.full_messages.to_sentence}" unless app.persisted?

    secret = app.plaintext_secret
    raise 'OAuth application secret was not available after create' if secret.blank?

    [app.uid, secret]
end

admin = User.find_by_username('root')
raise 'root user not found' if admin.nil?

org = organization
raise 'default organization not found' if org.nil?

ensure_local_webhooks(ApplicationSetting.current)
user = ensure_user(admin, org)
admin_token = ensure_token(admin, admin, org)
e2e_token = ensure_token(admin, user, org)
runner_token = ensure_runner(admin)
client_id, client_secret = ensure_oauth_application(org)

payload = {
    gitlab_url: 'http://gitlab.test:8929',
    client_id: client_id,
    client_secret: client_secret,
    admin_token: admin_token,
    e2e_token: e2e_token,
    runner_token: runner_token,
    e2e_user_id: user.id,
    e2e_username: user.username,
    e2e_password: E2E_PASSWORD,
    root_user_id: admin.id,
}
puts "E2E_CREDENTIALS_JSON=#{payload.to_json}"
