# Extend OpenAI account access to API keys

The administrator OpenAI OAuth access matrix currently lists and enforces
only OpenAI OAuth root accounts. Extend the same user/account policy to
OpenAI API-key root accounts so administrators can make an API-key account
public or restricted and grant it to selected local users.

The existing endpoint and persistence names remain compatible. Setup-token
accounts, other platforms, and account shadows remain outside the matrix.
Existing accounts without a policy remain public.
