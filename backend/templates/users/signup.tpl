{{ define "users/signup.tpl" }}
        <h2>Signup</h2>
        <form action="/users/signup" method="POST">
            <label for="email">Email</label><br>
            <input type="email" id="email" name="email" required><br>
            
            <label for="password">Password:</label><br>
            <input type="password" id="password" name="password" required><br>
            
            <label for="canvas">Canvas:</label><br>
            <input type="canvas" id="canvas" name="canvas" required><br><br>

            <input type="submit" value="Signup">
        </form>
{{ end }}
