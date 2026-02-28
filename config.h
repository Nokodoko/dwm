/* See LICENSE file for copyright and license details. */

#include <X11/XF86keysym.h>

/* appearance */
static const unsigned int borderpx  = 2;        /* border pixel of windows */
static const unsigned int gappx     = 2;        /* gap pixel between windows */
static const unsigned int snap      = 32;       /* snap pixel */
static const int showbar            = 1;        /* 0 means no bar */
static const int topbar             = 1;        /* 0 means bottom bar */
static const int user_bh            = 0;        /* 0 = font-based bar height, >0 = fixed px */
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

/* tagging */
static const char *tags[] = { ">_", "(-.x)", "~>", "4", "5", "6", "7", "8", "9", "SP", "SP2", "OLR", "AI", "STM", "SSH" };
#define SCRATCHPAD_TAG (1 << (LENGTH(tags) - 6))
#define BTOP_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 5))
#define OLR_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 4))
#define AI_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 3))
#define STEAM_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 2))
#define SSH_SCRATCHPAD_TAG (1 << (LENGTH(tags) - 1))

static const Rule rules[] = {
	/* xprop(1):
	 * 	WM_CLASS(STRING) = instance, class
	 * 	WM_NAME(STRING) = title
	 */
	/* class              instance  title  tags mask              isfloating  monitor  iscentered  bw  borderscheme  bordertitle */
    {"Gimp",              NULL,     NULL,  0,                     1,          -1,      0,          -1, -1,           NULL},
    {"wezterm-lf",        NULL,     NULL,  0,                     1,          -1,      1,           1, SchemeOlr,    "lf"},
    {"wezterm-tabtiler",  NULL,     NULL,  0,                     1,          -1,      1,           1, SchemeOlr,    "tiles"},
    {"dmenu-tabkill",     NULL,     NULL,  0,                     1,          -1,      1,           1, SchemeOlr,    "kill"},
    {"term-scratchpad",   NULL,     NULL,  SCRATCHPAD_TAG,        1,          -1,      0,          -1, -1,           NULL},
    {"btop-scratchpad",   NULL,     NULL,  BTOP_SCRATCHPAD_TAG,   1,          -1,      0,          -1, -1,           NULL},
    {"olr-scratchpad",    NULL,     NULL,  OLR_SCRATCHPAD_TAG,    1,          -1,      1,           1, SchemeOlr,    "olr"},
    {"ai-scratchpad",     NULL,     NULL,  AI_SCRATCHPAD_TAG,     1,          -1,      0,           1, SchemeAI,     "AI"},
    {"stm-scratchpad",    NULL,     NULL,  STEAM_SCRATCHPAD_TAG,  1,          -1,      1,           1, SchemeSteam,  "Steam"},
    {"ssh-scratchpad",    NULL,     NULL,  SSH_SCRATCHPAD_TAG,    1,          -1,      0,          -1, -1,           NULL},
    {"St",                    NULL,     NULL,  1,                     0,          0,       1,          -1, -1,           NULL},
    {"wireshark",             NULL,     NULL,  1,                     0,          0,       -1,         -1, -1,           NULL},
    {"Slack",                 NULL,     NULL,  1 << 7,                0,          -1,      0,          -1, -1,           NULL},
    {"Teams",                 NULL,     NULL,  1 << 1,                0,          -1,      0,          -1, -1,           NULL},
    {"mpv",                   NULL,     NULL,  1 << 2,                0,          -1,      0,          -1, -1,           NULL},
    {"firefox",               NULL,     NULL,  1 << 2,                0,          -1,      0,          -1, -1,           NULL},
    {"Vivaldi-flatpak",       NULL,     NULL,  1 << 2,                0,          -1,      0,          -1, -1,           NULL},
    {"Vivaldi-stable",        NULL,     NULL,  1 << 2,                0,          -1,      0,          -1, -1,           NULL},
    {"chromium",              NULL,     NULL,  1 << 2,                0,          -1,      0,          -1, -1,           NULL},
    {"qutebrowser",           NULL,     NULL,  1 << 6,                0,          -1,      0,          -1, -1,           NULL},
    {"Google Chrome",         NULL,     NULL,  1 << 3,                0,          -1,      0,          -1, -1,           NULL},
    {"Electron",              NULL,     NULL,  1 << 4,                0,          -1,      0,          -1, -1,           NULL},
    {"discord",               NULL,     NULL,  1 << 4,                0,          -1,      0,          -1, -1,           NULL},
    {"steam",                 NULL,     NULL,  1 << 6,                0,          -1,      0,          -1, -1,           NULL},

};

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
	{ MODKEY,                       KEY,      view,           {.ui = 1 << TAG} }, \
	{ MODKEY|ControlMask,           KEY,      toggleview,     {.ui = 1 << TAG} }, \
	{ MODKEY|ShiftMask,             KEY,      tag,            {.ui = 1 << TAG} }, \
	{ MODKEY|ControlMask|ShiftMask, KEY,      toggletag,      {.ui = 1 << TAG} },

/* helper for spawning shell commands in the pre dwm-5.0 fashion */
#define SHCMD(cmd) { .v = (const char*[]){ "/bin/sh", "-c", cmd, NULL } }

/* status bar process name for click actions */
#define STATUSBAR "dwmblocks"

/* commands */
static char dmenumon[2] = "0"; /* component of dmenucmd, manipulated in spawn() */
static const char *dmenucmd[] = { "dmenu_run", "-m", dmenumon, "-fn", dmenufont, "-nb", col_gray1, "-nf", col_gray3, "-sb", col_cyan, "-sf", col_gray4, NULL };
static const char *termcmd[]  = { "wezterm", NULL };
static const char *lyxcmd[]  = { "wezterm", "-e", "lyx", NULL };
static const char *killcmd[]  = { "killer.py", NULL };
static const char *pass[]  = { "pass.py", NULL };
static const char *hb[]  = { "hb.sh", NULL };
static const char *lb[]  = { "lb.sh", NULL };
static const char *pavucontrol[]  = { "pavucontrol", NULL };
static const char *dhp[]  = { "dhp.zsh", NULL };
static const char *mouseOn[]  = { "touchpadOn.lua", NULL };
static const char *mouseOff[]  = { "touchpadOff.lua", NULL };
static const char *volumeUp[]  = { "/home/n0ko/scripts/volume.sh", "up", NULL };
static const char *volumeDown[]  = { "/home/n0ko/scripts/volume.sh", "down", NULL };
static const char *volumeMute[]  = { "/home/n0ko/scripts/volume.sh", "mute", NULL };
static const char *cal[]  = { "wezterm", "-e", "calcurse", NULL };
static const char *top[]  = { "wezterm", "-e", "btop", NULL };
static const char *yazi[]  = { "/home/n0ko/scripts/fm-launcher.sh", "yazi", NULL };
static const char *scratchpadcmd[] = {"wezterm", "start", "--class", "term-scratchpad", NULL};
static const char *btopscratchpadcmd[] = {"wezterm", "start", "--class", "btop-scratchpad", "--", "btop", NULL};
static const char *olrscratchpadcmd[] = {"wezterm", "start", "--class", "olr-scratchpad", "--", "/usr/local/bin/olr", NULL};
static const char *aiscratchpadcmd[] = {"wezterm", "start", "--class", "ai-scratchpad", "--", "/home/n0ko/misc/hostlister.sh", NULL};
static const char *steamscratchpadcmd[] = {"wezterm", "start", "--class", "stm-scratchpad", "--", "/home/n0ko/scripts/steam_launcher.zsh", NULL};
static const char *sshscratchpadcmd[] = {"wezterm", "start", "--class", "ssh-scratchpad", "--", "ssh", "monty", NULL};
static const char *scrot_precision[] = { "/bin/sh", "-c", "scrot -s -e 'xclip -selection clipboard -t image/png -i $f && notify-send \"Screenshot Precision\" \"Copied to clipboard\"'", NULL };
static const char *slockcmd[] = { "/home/n0ko/scripts/slock-dpms.sh", NULL };
static const char *restartdwm[] = { "/home/n0ko/scripts/restart_dwm.sh", NULL };
static const char *restartdwm_wt[] = { "/home/n0ko/scripts/restart_dwm_worktree.sh", NULL };
static const char *brightnessUp[] = { "/home/n0ko/scripts/brightnessUp.sh", NULL };
static const char *brightnessDown[] = { "/home/n0ko/scripts/brightnessDown.sh", NULL };
static const char *brightnessMid[] = { "/home/n0ko/scripts/brightnessMid.sh", NULL };
static const char *xboxConnect[] = { "/home/n0ko/scripts/xbox.sh", NULL };
static const char *vivaldileadercmd[] = { "/home/n0ko/.local/bin/vivaldi-leader.sh", NULL };
static const char *dwmleadercmd[] = { "/home/n0ko/.local/bin/dwm-leader.sh", NULL };

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
	{ MODKEY,                       XK_r,      spawn,          {.v = restartdwm } },
	{ MODKEY|ShiftMask,             XK_r,      spawn,          {.v = restartdwm_wt } },
	{ MODKEY,                       XK_p,      spawn,          {.v = dmenucmd } },
	{ MODKEY,                       XK_x,      spawn,          {.v = lyxcmd } },
	{ MODKEY|ShiftMask,             XK_x,      spawn,          {.v = killcmd } },
	{ MODKEY|ShiftMask,             XK_c,      spawn,          {.v = cal } },
	{ MODKEY,                       XK_m,      spawn,          {.v = pavucontrol } },
	{ MODKEY,                       XK_o,      spawn,          {.v = mouseOff } },
	{ MODKEY|ShiftMask,             XK_o,      spawn,          {.v = mouseOn } },
	{ MODKEY|ControlMask,           XK_n,      spawn,          {.v = volumeUp } },
	{ MODKEY|ControlMask,           XK_b,      spawn,          {.v = volumeDown } },
	{ MODKEY|ControlMask,           XK_v,      spawn,          {.v = volumeMute } },
	{ MODKEY|ShiftMask,             XK_p,      spawn,          {.v = pass} },
	{ Mod1Mask|ShiftMask,           XK_t,      spawn,          {.v = top} },
  { Mod1Mask|ControlMask,         XK_Down,   spawn,          {.v = dhp } },
  { Mod4Mask|ControlMask,         XK_Right,  spawn,          {.v = hb } },
  { Mod4Mask|ControlMask,         XK_Left,   spawn,          {.v = lb } },
  { Mod1Mask|Mod4Mask,            XK_1,      spawn,          {.v = scrot_precision } },
  { Mod4Mask|ShiftMask,           XK_Return, spawn,          {.v = termcmd } },
	{ MODKEY|ShiftMask,             XK_l,      spawn,          {.v = yazi } },
	{ MODKEY,                       XK_b,      togglebar,      {0} },
	{ MODKEY,                       XK_j,      focusstack,     {.i = +1 } },
	{ MODKEY,                       XK_k,      focusstack,     {.i = -1 } },
	{ MODKEY,                       XK_i,      incnmaster,     {.i = +1 } },
	{ MODKEY,                       XK_d,      incnmaster,     {.i = -1 } },
	{ MODKEY,                       XK_h,      setmfact,       {.f = -0.05} },
	{ MODKEY,                       XK_l,      setmfact,       {.f = +0.05} },
	{ MODKEY|ControlMask,           XK_l,      spawn,          {.v = slockcmd } },
	{ 0,                            XF86XK_MonBrightnessUp,   spawn, {.v = brightnessUp } },
	{ 0,                            XF86XK_MonBrightnessDown, spawn, {.v = brightnessDown } },
	{ MODKEY|ControlMask,           XK_m,      spawn,          {.v = brightnessMid } },
	{ MODKEY,                       XK_Return, zoom,           {0} },
	{ MODKEY,                       XK_Tab,    view,           {0} },
	{ Mod4Mask|ShiftMask,           XK_q,      killclient,     {0} },
	{ MODKEY,                       XK_t,      setlayout,      {.v = &layouts[0]} },
	{ MODKEY,                       XK_f,      setlayout,      {.v = &layouts[1]} },
	{ MODKEY,                       XK_space,  setlayout,      {0} },
	{ MODKEY|ShiftMask,             XK_space,  togglefloating, {0} },
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
/* click can be ClkTagBar, ClkLtSymbol, ClkStatusText, ClkWinTitle, ClkClientWin, or ClkRootWin */
static const Button buttons[] = {
	/* click                event mask      button          function        argument */
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
