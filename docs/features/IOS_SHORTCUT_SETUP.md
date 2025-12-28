# iOS Shortcut Setup Guide

Unfortunately, iOS does not support PWA shortcuts defined in `manifest.json`. However, you can manually create a Siri Shortcut that works with FoodList.

## Manual Setup Instructions

### Step 1: Get Your App URL

First, note your FoodList app URL. It will look something like:
- `https://yourdomain.com/your-secret-path/`
- Or `https://yourdomain.com/` if you're using a whitelisted IP

### Step 2: Create a Siri Shortcut

1. **Open the Shortcuts app** on your iPhone (it's pre-installed)

2. **Tap the "+" button** in the top right to create a new shortcut

3. **Add an action:**
   - Search for "Open URLs" or "URL"
   - Add the "Open URLs" action

4. **Configure the URL:**
   - In the URL field, enter: `https://yourdomain.com/your-secret-path/?action=add&text=%s`
   - Replace `yourdomain.com/your-secret-path/` with your actual app URL
   - The `%s` will be replaced by Siri with the spoken text

5. **Name your shortcut:**
   - Tap "Next" in the top right
   - Name it "handla" (or whatever you prefer)
   - Tap "Done"

6. **Enable Siri:**
   - Tap the shortcut you just created
   - Tap "Add to Siri"
   - Record your phrase: "handla" (or "handla mjölk" to test)
   - Tap "Done"

### Step 3: Test It

Say "Hey Siri, handla mjölk" and Siri should:
1. Open the FoodList app
2. Extract "mjölk" from "handla mjölk"
3. Use autocomplete to find the best match
4. Add the item to your list

## How It Works

When you say "handla mjölk sirap potatis":

1. **Siri replaces `%s`** with your spoken text: `handla mjölk sirap potatis`
2. **The app opens** with URL: `?action=add&text=handla%20mjölk%20sirap%20potatis`
3. **The app extracts** the items after "handla": `mjölk sirap potatis`
4. **Smart splitting** matches known items first (e.g., if "mjölk sirap" exists, it matches that)
5. **Autocomplete** finds the best match for each item from your history
6. **Items are added** to your shopping list

## Tips

- **Use the exact phrase** you recorded when invoking Siri
- **Speak clearly** - Siri's transcription affects the results
- **Test with known items** - items you've added before will match better
- **Multi-word items** work! If you have "mjölk sirap" in your history, saying "handla mjölk sirap potatis" will match it correctly

## Troubleshooting

### Shortcut doesn't open the app
- Make sure you've added FoodList to your home screen (it needs to be installed as a PWA)
- Check that the URL in your shortcut matches your app's URL exactly

### Items aren't being added
- Make sure you're connected to the internet
- Check that the app has synced (look for the connection indicator)
- Try saying the items more clearly

### Wrong items are added
- The app uses autocomplete to find the best match
- If you want exact matches, make sure you say the exact item name
- Items you've used before will match better than new ones

## Alternative: Using Shortcuts App Directly

You can also create shortcuts that:
- Add specific items (without voice input)
- Combine multiple actions
- Work with other apps

Example: Create a shortcut called "Weekly Shopping" that opens FoodList with multiple items pre-filled.

