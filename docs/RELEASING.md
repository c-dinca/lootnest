# Releasing

A release is a merge to `main` with a new `VERSION` and a matching `## x.y.z`
section at the top of `CHANGELOG.md`. GitHub Actions (`.github/workflows/build.yml`)
builds `Lootnest.exe` and `Lootnest-Setup.exe` and publishes the release.

Two more steps are in the workflow but switched off until they're set up:
signing both .exe files with SignPath Foundation's certificate, so Windows
SmartScreen and antivirus programs stop flagging new releases over time, and
sending each release to winget, whose installs don't get the SmartScreen
prompt at all. These are the one-time steps for the repository's owner.

## 1. Code signing (SignPath Foundation)

SignPath Foundation signs open source projects for free. It needs builds to
come from GitHub Actions and the README's "Code signing policy" section,
which is already there.

1. Turn on two-factor authentication on your GitHub account (Settings,
   Password and authentication). SignPath requires it of everyone who can
   change the code or approve a signature.
2. Apply at https://signpath.org/apply for the Foundation programme, with
   this repository's URL. Wait for the answer; they review the project by hand.
3. Once accepted, in SignPath (https://app.signpath.io):
   - add the predefined Trusted Build System **GitHub.com** to the
     organization, and install the SignPath GitHub App on this repository;
   - create a project for Lootnest (its slug is, for example, `lootnest`)
     and link it to GitHub.com;
   - give it an artifact configuration that signs the .exe inside the zip
     GitHub makes of each upload:

     ```xml
     <?xml version="1.0" encoding="utf-8"?>
     <artifact-configuration xmlns="http://signpath.io/artifact-configuration/v1">
       <zip-file>
         <pe-file path="*.exe">
           <authenticode-sign/>
         </pe-file>
       </zip-file>
     </artifact-configuration>
     ```

   - create a signing policy (for example `release-signing`) with the
     Foundation's certificate, and yourself as approver;
   - create an API token for a user who may submit to that policy.
4. In this repository on GitHub, Settings, Secrets and variables, Actions:
   - Secrets: `SIGNPATH_API_TOKEN`, the API token;
   - Variables: `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG` and
     `SIGNPATH_POLICY_SLUG`, from SignPath.
5. Add the variable `SIGNPATH_ENABLED` with the value `true`.

From then on, every build that becomes a release sends `Lootnest.exe` for
signing, puts the signed one into the setup, then sends `Lootnest-Setup.exe`.
Pull requests are never signed. If the policy asks for approval, approve each
request in SignPath while the workflow waits. It waits an hour for each of the
two; a request not approved by then fails the build, and no release is
published. Re-run the workflow from the Actions tab (or push to `main` again
while the version has no tag yet) to send it again. To stop signing, set
`SIGNPATH_ENABLED` to anything else.

## 2. winget

1. Fill in `InstallerSha256` in `packaging/winget/CDinca.Lootnest.installer.yaml`
   from the release (GitHub shows the SHA-256 next to `Lootnest-Setup.exe`, or
   run `winget hash Lootnest-Setup.exe`), and set the version in all three
   files to that release's.
2. Fork https://github.com/microsoft/winget-pkgs to your account, add the
   three files under `manifests/c/CDinca/Lootnest/<version>/`, and open a pull
   request. Its checks install Lootnest with `--silent` on a clean Windows.
   Wait until it is merged.
3. Create a classic personal access token (Settings, Developer settings,
   Personal access tokens, Tokens (classic)) with the `public_repo` scope
   (and `workflow`, so the fork can be kept in sync), and save it as the
   repository secret `WINGET_TOKEN`.
4. Add the repository variable `WINGET_ENABLED` with the value `true`.

From then on, each new release opens the winget pull request by itself.

## 3. The website

The website in `site/` is published by `.github/workflows/site.yml`. It
needs GitHub Pages turned on once: Settings, Pages, Source: **GitHub Actions**.
