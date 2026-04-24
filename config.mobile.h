// TODO: remove config.multi.h once mobile is stable as daily driver.
/* See LICENSE file for copyright and license details. */

#include <X11/XF86keysym.h>

/* appearance */
static const unsigned int borderpx  = 2;        /* border pixel of windows */
static const unsigned int gappx     = 2;        /* gap pixel between windows */
static const unsigned int snap      = 32;       /* snap pixel */
static const int showbar            = 1;        /* 0 means no bar */
static const int topbar             = 1;        /* 0 means bottom bar */
static const char *fonts[] = {"VictorMono Nerd Font Mono:style=Italic:size=11"};
static const char dmenufont[] = "VictorMono Nerd Font Mono:style=Italic:size=11";
static const char col_gray1[] = "#000000";
static const char col_gray2[] = "#333333";
static const char col_gray3[] = "#5F5F00";
static const char col_gray4[] = "#00FF00";
static const char col_cyan[] = "#000000";
static const char col_lightblue[] = "#ADD8E6";
static const char col_teal[] = "#0088AA";
static const char col_green[] = "#00AA44";
static const char col_purple[] = "#9932CC";
static const char *colors[][3]      = {
	/*               fg         bg         border   */
	[SchemeNorm] = { col_gray3, col_gray1, col_gray2 },
	[SchemeSel]  = { col_gray4, col_cyan,  col_lightblue  },
	[SchemeOlr]  = { col_gray3, col_gray1, col_teal  },
	[SchemeAI]   = { col_gray3, col_gray1, col_green },
	[SchemeSteam] = { col_gray3, col_gray1, col_purple },
};

/* tagging — nerd font icons: web, chat, term, team, rocket, code, game, slack, music */
static const char *tags[] = { "\xef\x82\xac", "\xef\x81\xb5", "\xef\x92\x89", "\xef\x83\x80", "\xef\x84\xb5", "\xef\x84\xa1", "\xef\x84\x9b", "\xef\x86\x98", "\xef\x80\x81", "SP", "SP2", "OLR", "AI", "STM", "SSH" };
#define SCRATCHPAD_TAG (1 << (LENGTH(tags) - 6))
#define BTOP_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 5))
#define OLR_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 4))
#define AI_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 3))
#define STEAM_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 2))
#define SSH_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 1))

static const Rule rules[] = {
	/* xprop(1):
	 *   WM_CLASS(STRING) = instance, class
	 *   WM_NAME(STRING) = title
	 *
	 * Mobile (2-monitor) pertag rules (z13 undocked + external strip):
	 *   Mon 0 (eDP-1, 2560x1600):    primary work area, tags 3-9
	 *     - Terminals (>_) -> tag 3 (1<<2)
	 *     - Team/Agents    -> tag 4 (1<<3)
	 *     - Monitoring     -> tag 5 (1<<4)
	 *     - Code/Agents    -> tag 6 (1<<5)
	 *     - Games          -> tag 7 (1<<6)
	 *     - Slack          -> tag 8 (1<<7)
	 *     - Media          -> tag 9 (1<<8)
	 *   Mon 1 (DP-1, 2560x720 strip): secondary / chat strip, tags 1-2
	 *     - Browsers (www) -> tag 1 (1<<0)
	 *     - Chat (cht)     -> tag 2 (1<<1)
	 *   Scratchpads: floating, monitor -1 (follow focus)
	 */
	/* class              instance  title           tags mask              isfloating  monitor  iscentered  bw  borderscheme  bordertitle  floatw  floath */

	/* --- Floating utilities (follow focus) --- */
	{"Gimp",              NULL,     NULL,           0,                     1,          -1,      0,          -1, -1,           NULL,        1200,   900},

	/* --- Scratchpads: floating, follow focus (mon -1) --- */
	{"term-scratchpad",   NULL,     NULL,           SCRATCHPAD_TAG,        1,          -1,      0,          -1, -1,           NULL,        1200,   900},
	{"btop-scratchpad",   NULL,     NULL,           BTOP_SCRATCHPAD_TAG,   1,          -1,      0,          -1, -1,           NULL,        1200,   900},
	{"olr-scratchpad",    NULL,     NULL,           OLR_SCRATCHPAD_TAG,    1,          -1,      1,           1, SchemeOlr,    "olr",       1200,   900},
	{"ai-scratchpad",     NULL,     NULL,           AI_SCRATCHPAD_TAG,     1,          -1,      0,           1, SchemeAI,     "AI",        1200,   900},
	{"stm-scratchpad",    NULL,     NULL,           STEAM_SCRATCHPAD_TAG,  1,          -1,      1,           1, SchemeSteam,  "Steam",     1200,   900},
	{"ssh-scratchpad",    NULL,     NULL,           SSH_SCRATCHPAD_TAG,    1,          -1,      0,          -1, -1,           NULL,        1200,   900},

	/* --- Floating overlays (follow focus) --- */
	{"kitty-lf",          NULL,     NULL,           0,                     1,          -1,      1,           1, SchemeOlr,    "lf",        1200,   900},
	{"kitty-lister",      NULL,     NULL,           0,                     1,          -1,      1,           1, SchemeOlr,    "lister",    1100,   650},
	{"kitty-tabtiler",    NULL,     NULL,           0,                     1,          -1,      1,           1, SchemeOlr,    "tiles",     1200,   900},
	{"gpu-select",        NULL,     NULL,           0,                     1,          -1,      1,           1, SchemeOlr,    "gpu",        800,   500},
	{"trustgraph-viewer", NULL,     NULL,           0,                     1,          -1,      0,          -1, -1,           NULL,        0,      0},

	/* --- Mon 1 (DP-1, strip): Tag 1 (browsers, 1<<0) --- */
	{"firefox",               NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"Vivaldi-stable",        NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"Vivaldi-flatpak",       NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"chromium",              NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"qutebrowser",           NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"Google-chrome",         NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},

	/* --- Mon 1 (DP-1, strip): Tag 2 (chat, 1<<1) --- */
	{"teams-for-linux",       NULL,     NULL,       1 << 1,            0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"Slack",                 NULL,     NULL,       1 << 1,            0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"discord",               NULL,     NULL,       1 << 1,            0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"ZapZap",                NULL,     NULL,       1 << 1,            0,           1,      0,          -1, -1,           NULL,        0,      0},
	{"Electron",              NULL,     NULL,       1 << 1,            0,           1,      0,          -1, -1,           NULL,        0,      0},

	/* --- Mon 0 (eDP-1): Tag 3 (terminals, 1<<2) --- */
	{"St",                    NULL,     NULL,       1 << 2,            0,           0,      1,          -1, -1,           NULL,        0,      0},
	{"kitty",                 NULL,     NULL,       1 << 2,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"neovide",               NULL,     NULL,       1 << 2,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"wireshark",             NULL,     NULL,       1 << 2,            0,           0,      -1,         -1, -1,           NULL,        0,      0},

	/* --- Mon 0 (eDP-1): Tag 4 (team/agents, 1<<3) --- */
	{"cmdr-dashboard",        NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"cmdr-terminal",         NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"overstory-terminal",    NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"cmdr-feed",             NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"cmdr-costs",            NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"cmdr-logs",             NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},
	{"cmdr-errors",           NULL,     NULL,       1 << 3,            0,           0,      0,          -1, -1,           NULL,        0,      0},

	/* --- Mon 0 (eDP-1): games (1<<6) --- */
	{"steam",                 NULL,     NULL,       1 << 6,            0,           0,      0,          -1, -1,           NULL,        0,      0},

	/* --- Mon 0 (eDP-1): media (1<<8) --- */
	{"mpv",                   NULL,     NULL,       1 << 8,            0,           0,      0,          -1, -1,           NULL,        0,      0},

	/*
	 * Trustgraph / localhost:3000 — use Vivaldi app mode.
	 */
	{NULL,                    NULL,     "trustgraph", 0,               1,          -1,      1,          -1, -1,           NULL,        1200,   900},
	{NULL,                    NULL,     "localhost",  0,               1,          -1,      1,          -1, -1,           NULL,        1200,   900},
};

/* default tags per monitor (index = monitor number) */
static const unsigned int defaulttags[] = {
    1 << 2,   /* mon 0 (eDP-1):              tag 3 (term) */
    1 << 0,   /* mon 1 (DP-1, 2560x720):     tag 1 (www)  */
};

/* tag-to-monitor map: which monitor owns each tag (index = tag index) */
static const int tagmonmap[] = { 1, 1, 0, 0, 0, 0, 0, 0, 0 };
/*                                ^  ^  ^  ^  ^  ^  ^  ^  ^
 *                           tag: 1  2  3  4  5  6  7  8  9
 *                          icon: ww ch >_ tm rk cd gm sl mu
 *                           mon: 1  1  0  0  0  0  0  0  0
 *                          name: DP1 DP1 eDP eDP eDP eDP eDP eDP eDP */

/* layout(s) */
const float mfact     = 0.55; /* factor of master area size [0.05..0.95] */
static const int nmaster     = 1;    /* number of clients in master area */
static const int resizehints = 1;    /* 1 means respect size hints in tiled resizals */
static const int lockfullscreen = 1; /* 1 will force focus on the fullscreen window */

static const Layout layouts[] = {
	/* symbol     arrange function */
	{ "[]=",      tile },    /* first entry is default */
	{ "[M]",      monocle },
	{ "><>",      NULL },    /* no layout function means floating behavior */
};

/* key definitions */
#define MODKEY Mod4Mask
#define TAGKEYS(KEY,TAG) \
	{ MODKEY,                       KEY,      viewmon,        {.ui = 1 << TAG} }, \
	{ MODKEY|ControlMask,           KEY,      toggleview,     {.ui = 1 << TAG} }, \
	{ MODKEY|ShiftMask,             KEY,      tag,            {.ui = 1 << TAG} }, \
	{ MODKEY|ControlMask|ShiftMask, KEY,      toggletag,      {.ui = 1 << TAG} },

/* helper for spawning shell commands in the pre dwm-5.0 fashion */
#define SHCMD(cmd) { .v = (const char*[]){ "/bin/sh", "-c", cmd, NULL } }

/* status bar process name for click actions */
#define STATUSBAR "dwmblocks"

/* commands */
static char dmenumon[2] = "0"; /* component of dmenucmd, single monitor */
static const char *dmenucmd[] = { "/home/n0ko/scripts/lister.sh", NULL };
static const char *termcmd[]  = { "kitty", NULL };
static const char *lyxcmd[]  = { "kitty", "--", "lyx", NULL };
static const char *killcmd[]  = { "killer.py", NULL };
static const char *pass[]  = { "pass.py", NULL };
static const char *wificmd[]  = { "wifi.py", NULL };
static const char *hb[]  = { "hb.sh", NULL };
static const char *lb[]  = { "lb.sh", NULL };
static const char *pavucontrol[]  = { "pavucontrol", NULL };
static const char *dhp[]  = { "dhp.zsh", NULL };
static const char *mouseOn[]  = { "touchpadOn.sh", NULL };
static const char *mouseOff[]  = { "touchpadOff.sh", NULL };
static const char *volumeUp[]  = { "/home/n0ko/scripts/volume.sh", "up", NULL };
static const char *volumeDown[]  = { "/home/n0ko/scripts/volume.sh", "down", NULL };
static const char *volumeMute[]  = { "/home/n0ko/scripts/volume.sh", "mute", NULL };
static const char *cal[]  = { "kitty", "--", "calcurse", NULL };
static const char *top[]  = { "kitty", "--", "btop", NULL };
static const char *yazi[]  = { "/home/n0ko/scripts/fm-launcher.sh", "yazi", NULL };
static const char *scratchpadcmd[] = {"/bin/neovide", "--frame=none", "--x11-wm-class", "term-scratchpad", "--x11-wm-class-instance=term-scratchpad", NULL};
static const char *btopscratchpadcmd[] = {"kitty", "--class", "btop-scratchpad", "--", "btop", NULL};
static const char *olrscratchpadcmd[] = {"kitty", "--class", "olr-scratchpad", "--", "/usr/local/bin/olr", NULL};
static const char *aiscratchpadcmd[] = {"kitty", "--class", "ai-scratchpad", "--", "/home/n0ko/misc/hostlister.sh", NULL};
static const char *steamscratchpadcmd[] = {"kitty", "--class", "stm-scratchpad", "--", "/home/n0ko/scripts/steam_launcher.zsh", NULL};
static const char *sshscratchpadcmd[] = {"kitty", "--class", "ssh-scratchpad", "--", "ssh", "-t", "base", "zellij", "attach", "-c", "default", NULL};
static const char *scrot_precision[] = { "/bin/sh", "-c", "scrot -s -e 'xclip -selection clipboard -t image/png -i $f && notify-send \"Screenshot Precision\" \"Copied to clipboard\"'", NULL };
static const char *slockcmd[] = { "/home/n0ko/scripts/slock-dpms.sh", NULL };
static const char *tgsnapcmd[] = { "/usr/local/bin/tg-snap", NULL };
static const char *lewislayoutcmd[] = { "/home/n0ko/scripts/lewis-layout.sh", NULL };
static const char *restartdwm[] = { "/home/n0ko/scripts/dwm-hotswap.sh", "pertag", NULL };
static const char *restartdwm_wt[] = { "/home/n0ko/scripts/dwm-hotswap.sh", "pertag-multi", NULL };
static const char *restartdwm_base[] = { "/home/n0ko/scripts/dwm-hotswap.sh", "base", NULL };
static const char *brightnessUp[] = { "/home/n0ko/scripts/brightnessUp.sh", NULL };
static const char *brightnessDown[] = { "/home/n0ko/scripts/brightnessDown.sh", NULL };
static const char *brightnessMid[] = { "/home/n0ko/scripts/brightnessMid.sh", NULL };
static const char *keybrightnessUp[] = { "/home/n0ko/scripts/keybrightnessUp.sh", NULL };
static const char *keybrightnessDown[] = { "/home/n0ko/scripts/keybrightnessDown.sh", NULL };
static const char *xboxConnect[] = { "/home/n0ko/scripts/xbox.sh", NULL };
static const char *vivaldileadercmd[] = { "/home/n0ko/.local/bin/vivaldi-leader.sh", NULL };
static const char *dwmleadercmd[] = { "/home/n0ko/.local/bin/dwm-leader.sh", NULL };

/* moveresize direction vectors: {dx, dy, dw, dh} */
static const int moveresizedelta_up[]    = {  0, -25,   0,   0 };
static const int moveresizedelta_down[]  = {  0,  25,   0,   0 };
static const int moveresizedelta_left[]  = { -25,  0,   0,   0 };
static const int moveresizedelta_right[] = {  25,  0,   0,   0 };
static const int moveresizedelta_sh[]    = {  0,   0,   0, -25 }; /* shrink height */
static const int moveresizedelta_gh[]    = {  0,   0,   0,  25 }; /* grow height */
static const int moveresizedelta_sw[]    = {  0,   0, -25,   0 }; /* shrink width */
static const int moveresizedelta_gw[]    = {  0,   0,  25,   0 }; /* grow width */

static const Key keys[] = {
	/* modifier                     key        function        argument */
	{ Mod1Mask,                     XK_s,      togglescratch,  {.v = scratchpadcmd } },
	{ Mod1Mask,                     XK_b,      togglescratch,  {.v = btopscratchpadcmd } },
	{ Mod1Mask,                     XK_o,      togglescratch,  {.v = olrscratchpadcmd } },
	{ Mod1Mask,                     XK_a,      togglescratch,  {.v = aiscratchpadcmd } },
        { Mod1Mask,                     XK_r,      togglescratch,  {.v = steamscratchpadcmd } },
	{ Mod1Mask|ShiftMask,           XK_s,      togglescratch,  {.v = sshscratchpadcmd } },
	{ Mod1Mask|ControlMask,         XK_v,      spawn,          {.v = vivaldileadercmd } },
	{ Mod1Mask,                     XK_t,      spawn,          {.v = dwmleadercmd } },
	{ Mod1Mask,                     XK_c,      spawn,          {.v = xboxConnect } },
	{ MODKEY,                       XK_F5,     spawn,          {.v = tgsnapcmd } },
	{ MODKEY,                       XK_r,      spawn,          {.v = restartdwm } },
	{ MODKEY|ShiftMask,             XK_r,      spawn,          {.v = restartdwm_wt } },
	{ MODKEY|ControlMask,           XK_r,      spawn,          {.v = restartdwm_base } },
	{ MODKEY|ControlMask,           XK_u,      spawn,          {.v = lewislayoutcmd } },
	{ MODKEY,                       XK_p,      spawn,          {.v = dmenucmd } },
	{ MODKEY,                       XK_x,      spawn,          {.v = lyxcmd } },
	{ MODKEY|ShiftMask,             XK_x,      spawn,          {.v = killcmd } },
	{ MODKEY|ShiftMask,             XK_c,      spawn,          {.v = cal } },
	{ MODKEY,                       XK_m,      spawn,          {.v = pavucontrol } },
	{ MODKEY,                       XK_o,      spawn,          {.v = mouseOff } },
	{ MODKEY|ShiftMask,             XK_o,      spawn,          {.v = mouseOn } },
	{ 0,                            XF86XK_AudioRaiseVolume, spawn, {.v = volumeUp } },
	{ 0,                            XF86XK_AudioLowerVolume, spawn, {.v = volumeDown } },
	{ 0,                            XF86XK_AudioMute,        spawn, {.v = volumeMute } },
	{ MODKEY|ShiftMask,             XK_p,      spawn,          {.v = pass} },
	{ MODKEY|ShiftMask,             XK_w,      spawn,          {.v = wificmd} },
	{ Mod1Mask|ShiftMask,           XK_t,      spawn,          {.v = top} },
  { Mod1Mask|ControlMask,         XK_Down,   spawn,          {.v = dhp } },
  { Mod4Mask|ControlMask,         XK_Right,  spawn,          {.v = hb } },
  { Mod4Mask|ControlMask,         XK_Left,   spawn,          {.v = lb } },
  { Mod1Mask,                     XK_1,      spawn,          {.v = scrot_precision } },
  { Mod4Mask|ShiftMask,           XK_Return, spawn,          {.v = termcmd } },
	{ MODKEY|ShiftMask,             XK_l,      spawn,          {.v = yazi } },
	{ MODKEY,                       XK_b,      togglebar,      {0} },
	{ MODKEY,                       XK_j,      focusstack,     {.i = +1 } },
	{ MODKEY,                       XK_k,      focusstack,     {.i = -1 } },
	{ MODKEY,                       XK_i,      incnmaster,     {.i = +1 } },
	{ MODKEY,                       XK_d,      incnmaster,     {.i = -1 } },
	{ MODKEY,                       XK_u,      incnmaster,     {.i = 0 } },
	{ MODKEY,                       XK_h,      setmfact,       {.f = -0.05} },
	{ MODKEY,                       XK_l,      setmfact,       {.f = +0.05} },
	{ MODKEY|ControlMask,           XK_l,      spawn,          {.v = slockcmd } },
	{ 0,                            XF86XK_MonBrightnessUp,   spawn, {.v = brightnessUp } },
	{ 0,                            XF86XK_MonBrightnessDown, spawn, {.v = brightnessDown } },
	{ MODKEY|ControlMask,           XK_m,      spawn,          {.v = brightnessMid } },
	{ Mod1Mask|ControlMask,         XK_u,      spawn,          {.v = brightnessUp } },
	{ Mod1Mask|ControlMask,         XK_i,      spawn,          {.v = brightnessDown } },
	{ Mod1Mask|ControlMask,         XK_o,      spawn,          {.v = keybrightnessUp } },
	{ Mod1Mask|ControlMask,         XK_p,      spawn,          {.v = keybrightnessDown } },
	{ MODKEY,                       XK_Return, zoom,           {0} },
	{ MODKEY,                       XK_Tab,    view,           {0} },
	{ Mod4Mask|ShiftMask,           XK_q,      killclient,     {0} },
	{ MODKEY,                       XK_t,      setlayout,      {.v = &layouts[0]} },
	{ MODKEY,                       XK_f,      setlayout,      {.v = &layouts[1]} },
	{ MODKEY,                       XK_space,  setlayout,      {0} },
	{ MODKEY|ShiftMask,             XK_space,  togglefloating, {0} },
	{ MODKEY,                       XK_Up,     moveresize,     {.v = moveresizedelta_up } },
	{ MODKEY,                       XK_Down,   moveresize,     {.v = moveresizedelta_down } },
	{ MODKEY,                       XK_Left,   moveresize,     {.v = moveresizedelta_left } },
	{ MODKEY,                       XK_Right,  moveresize,     {.v = moveresizedelta_right } },
	{ MODKEY|ShiftMask,             XK_Up,     moveresize,     {.v = moveresizedelta_gh } },
	{ MODKEY|ShiftMask,             XK_Down,   moveresize,     {.v = moveresizedelta_sh } },
	{ MODKEY|ShiftMask,             XK_Left,   moveresize,     {.v = moveresizedelta_sw } },
	{ MODKEY|ShiftMask,             XK_Right,  moveresize,     {.v = moveresizedelta_gw } },
	{ MODKEY,                       XK_0,      view,           {.ui = ~0 } },
	{ MODKEY|ShiftMask,             XK_0,      tag,            {.ui = ~0 } },
	{ MODKEY,                       XK_comma,  focusmon,       {.i = -1 } },
	{ MODKEY,                       XK_period, focusmon,       {.i = +1 } },
	{ MODKEY|ShiftMask,             XK_comma,  tagmon,         {.i = -1 } },
	{ MODKEY|ShiftMask,             XK_period, tagmon,         {.i = +1 } },
	TAGKEYS(                        XK_1,                      0)
	TAGKEYS(                        XK_2,                      1)
	TAGKEYS(                        XK_3,                      2)
	TAGKEYS(                        XK_4,                      3)
	TAGKEYS(                        XK_5,                      4)
	TAGKEYS(                        XK_6,                      5)
	TAGKEYS(                        XK_7,                      6)
	TAGKEYS(                        XK_8,                      7)
	TAGKEYS(                        XK_9,                      8)
	{ MODKEY|ShiftMask,             XK_q,      quit,           {0} },
};

/* button definitions */
static const Button buttons[] = {
	{ ClkLtSymbol,          0,              Button1,        setlayout,      {0} },
	{ ClkLtSymbol,          0,              Button3,        setlayout,      {.v = &layouts[2]} },
	{ ClkWinTitle,          0,              Button2,        zoom,           {0} },
	{ ClkStatusText,        0,              Button1,        sigstatusbar,   {.i = 1} },
	{ ClkStatusText,        0,              Button2,        sigstatusbar,   {.i = 2} },
	{ ClkStatusText,        0,              Button3,        sigstatusbar,   {.i = 3} },
	{ ClkStatusText,        ShiftMask,      Button1,        sigstatusbar,   {.i = 6} },
	{ ClkClientWin,         MODKEY,         Button1,        movemouse,      {0} },
	{ ClkClientWin,         MODKEY,         Button2,        togglefloating, {0} },
	{ ClkClientWin,         MODKEY,         Button3,        resizemouse,    {0} },
	{ ClkTagBar,            0,              Button1,        view,           {0} },
	{ ClkTagBar,            0,              Button3,        toggleview,     {0} },
	{ ClkTagBar,            MODKEY,         Button1,        tag,            {0} },
	{ ClkTagBar,            MODKEY,         Button3,        toggletag,      {0} },
};
