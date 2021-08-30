# frozen_string_literal: true

set :stage, :staging
set :public_url, 'https://ex-stage.bitzlato.bz/'
set :build_domain, 'ex-stage.bitzlato.bz'

server ENV.fetch( 'STAGING_SERVER' ), user: fetch(:user), roles: fetch(:roles)
