# frozen_string_literal: true

lock '3.16'

set :user, 'app'
set :application, 'rango'

set :roles, %w[app].freeze

set :repo_url, ENV.fetch('DEPLOY_REPO', `git remote -v | grep origin | head -1 | awk  '{ print $2 }'`.chomp) if ENV['USE_LOCAL_REPO'].nil?
set :keep_releases, 10

set :deploy_to, -> { "/home/#{fetch(:user)}/#{fetch(:application)}" }

set :linked_files, %w[.env]
set :linked_dirs, %w[log]
set :config_files, fetch(:linked_files)

set :disallow_pushing, true

set :db_dump_extra_opts, '--force'

default_branch = 'main'
current_branch = `git rev-parse --abbrev-ref HEAD`.chomp

if ENV.key? 'BRANCH'
  set :branch, ENV.fetch('BRANCH')
elsif default_branch == current_branch
  set :branch, default_branch
else
  ask(:branch, proc { `git rev-parse --abbrev-ref HEAD`.chomp })
end

# set :app_version, SemVer.find.to_s
set :current_version, `git rev-parse HEAD`.strip

set :default_env, { }
# Use env from .env
# export JWT_PUBLIC_KEY=$(cat config/secrets/rsa-key.pub| base64 -w0)
# export RANGER_PORT=8081
# export LOG_LEVEL=debug
# export API_CORS_ORIGINS=dapi.bitzlato.bz,peatio.brandymint.ru,localhost:3000,localhost:3443,*

after 'deploy:updated', 'go_build'

set :goenv_type, :user # or :system, depends on your goenv setup
set :goenv_go_version, File.read('.go-version').chomp

task :go_build do
  on roles('app') do
    within release_path do
      execute :cp, 'go.mod', 'go.sum'
      execute :go, :mod, :download
      execute :go, :build, './cmd/rango'
    end
  end
end
if defined? Slackistrano
  Rake::Task['deploy:starting'].prerequisites.delete('slack:deploy:starting')
  set :slackistrano,
      klass: Slackistrano::CustomMessaging,
      channel: ENV['SLACKISTRANO_CHANNEL'],
      webhook: ENV['SLACKISTRANO_WEBHOOK']

  # best when 75px by 75px.
  set :slackistrano_thumb_url, 'https://bitzlato.com/wp-content/uploads/2020/12/logo.svg'
  set :slackistrano_footer_icon, 'https://github.githubassets.com/images/modules/logos_page/Octocat.png'
end

set :dotenv_hook_commands, %w{go}

after 'deploy:publishing', 'systemd:rango:reload-or-restart'
