{{ define "courses/index.tpl" }}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Canvas Courses</title>
</head>
<body>
    <h1>Your Quests</h1>

    <h2>Courses:</h2>
    {{ if .courses }}
        <ul>
            {{ range .courses }}
                <li>{{ .ID }} - {{ .Name }}</li>
            {{ else }}
                <li>No courses found.</li>
            {{ end }}
        </ul>
    {{ else }}
        <p>No courses available or an error occurred while fetching courses.</p>
    {{ end }}
</body>
</html>
{{ end }}
