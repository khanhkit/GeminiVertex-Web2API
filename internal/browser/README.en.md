# Chrome Cookie Fetcher Documentation

Batch retrieve Google cookies from multiple Chrome profiles.

## Usage

```bash
# 1. Close Chrome browser
# 2. Run the command
.\Gemini-Web2API.exe --fetch-cookies

# 3. Press Enter to continue

# 4. Select profiles
# Enter numbers (comma-separated): 1,2,3
# Or enter ALL to fetch all profiles
```

## Cookie Storage Format

```env
# Default account (no suffix)
__Secure-1PSID=xxx
__Secure-1PSIDTS=xxx

# Other accounts (uses the actual profile folder name as suffix)
__Secure-1PSID_niniro=xxx
__Secure-1PSIDTS_niniro=xxx
```

## Features

- **Concurrent Processing** - Fast parallel processing of multiple profiles
- **Auto-Retry** - Retries up to 3 times automatically on failure
- **Ordered Preservation** - Saves to `.env` in the selected order
- **Overwrite Mode** - Clears all old cookies, retaining only the newly fetched ones
- **Real Names** - Displays the actual names of Chrome profiles

## Notes

1. **Chrome must be closed** - Ensure Chrome is completely closed before running.
2. **Overwriting cookies** - This action deletes all existing `__Secure-1PSID*` variables in `.env`, keeping only the newly retrieved ones.
3. **Preserving other configs** - Other settings like `ACCOUNTS`, `PORT`, etc., remain untouched.
4. **Login required** - Each profile must have previously logged into gemini.google.com.

## How It Works

Powered by the Chrome DevTools Protocol (CDP):
1. Starts Chrome in headless mode.
2. Navigates to gemini.google.com.
3. Invokes `Network.getAllCookies` to retrieve all cookies.
4. Extracts `__Secure-1PSID` and `__Secure-1PSIDTS`.
5. Saves them to the `.env` file in the correct order.
