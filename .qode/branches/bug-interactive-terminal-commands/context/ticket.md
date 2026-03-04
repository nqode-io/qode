# BUG: Commands run in terminal are not interactive

When the user used the qode commands in the terminal they are not interactive and instead they run in the background.

What should happen instead is that claude code should run in the terminal taking over the process.

Special attention should be paid to `qode review all` as this one needs to run one claude code instance for code review, then after this finishes another for security review.
