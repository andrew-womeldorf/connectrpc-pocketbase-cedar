# Exprimenting with ConnectRPC, Pocketbase, and Cedar

An experiment combining [PocketBase](https://pocketbase.io/) as an embedded
backend with [ConnectRPC](https://connectrpc.com/) services, using
[Cedar](https://www.cedarpolicy.com/) for authorization.

The goal is to learn Cedar's policy language and toolchain while exploring what
it looks like to embed PocketBase and a ConnectRPC API in the same Go binary.

---

...And then in the process, i decided to also review authentication...

## Authentication Cheat Sheet

https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html

### User IDs

> Ideally, User IDs should be randomly generated to prevent the creation of
> predictable or sequential IDs...

pocketbase already assigns distinct ids. example: `lpc51qhlbqao4fb`

they're random:
https://github.com/pocketbase/pocketbase/blob/v0.40.3/core/field_text.go#L348-L369

### Usernames

> Users should be permitted to use their email address as a username, provided
> the email is verified during sign-up.

this is the default in pocketbase, including verification. however, it's the
responsibility of app code to check if the user is verified.

> Additionally, they should have the option to choose a username other than an
> email address

supported in pocketbase, but i haven't done it. should i?

TODO: I'm curious about changing email address, verification for people who
claim to have lost access to their email address, other validations, etc...

### Authentication Solution and Sensitive Accounts

> Do NOT allow login with sensitive accounts (i.e. accounts that can be used
> internally within the solution such as to a backend / middleware / database)
> to any front-end user interface.

technically possible to access the superuser account from the frontend. `cli
superuser ips` allows setting ip whitelist for accessing the superuser. i set
to localhost only, which means must be on the same machine... I wonder if
that's settable as part of the db migration? or some other startup mechanism...

> Do NOT use the same authentication solution (e.g. IDP / AD) used internally
> for unsecured access (e.g., public access / DMZ)

technically, yeah, it's different in the sense that `_superusers` and `users`
are different tables. i guess?

### Implement Proper Password Strength Controls

> - If MFA is enabled passwords shorter than 8 characters are considered to be
>   weak (NIST SP800-63B).
> - If MFA is not enabled passwords shorter than 15 characters are considered
>   to be weak (NIST SP800-63B).

by default, pocketbase puts the password minimum to 8 characters, but MFA is
disabled by default.

I've updated the migrations to enable OTP (which only has an email option) and
MFA. So now it's required. Therefore, an 8 character password is sufficient.

> Maximum password length should be at least 64 characters to allow passphrases
> (NIST SP800-63B). Note that certain implementations of hashing algorithms may
> cause long password denial of service.

default in pocketbase is 71 characters. that's fine.

> Do not silently truncate passwords

we don't. what's the point of min/max?

...oh, i see. they mean when the provided password is longer than what a hash algo can handle.

In the OWASP Password Storage Cheat sheet, they say *For legacy systems using bcrypt, use a work factor of 10 or more and with a password limit of 72 bytes.*. And it looks like, at a glance, pocketbase is using bcrypt. so that kinda makes sense.

> Allow usage of all characters including unicode and whitespace

This is the default in pocketbase, but it appears to be configurable. don't change it.

> Ensure credential rotation when a password leak occurs, at the time of compromise identification or when authenticator technology changes.

hmm. I wonder how to do this "properly". I can change the user's password for them...

> Include a password strength meter to help users create a more complex password

TODO: that requires javascript... which we don't have right now. maybe come back to this someday.

note that the guide offers some libraries to use. The specific library they mention is typescript...

> Block common and previously breached passwords

TODO: do this. there's links to the haveibeenpwned api, as well as where to download the database directly.

### Implement Secure Password Recovery Mechanism

this seems silly, with email otp... unless there's a way to do this without email.

TODO: see https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html

### Compare Password Hashes Using Safe Functions

This is up to pocketbase to do...

[yep, it does](https://github.com/pocketbase/pocketbase/blob/v0.40.3/core/field_password.go#L322)

### Change Password Feature

yep, we did it.

## Password Storage Cheat Sheet

https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html

TODO: review

I mean, pocketbase will do this for us. But I'm curious to look through how pocketbase does this compared to OWASP recommendations.
